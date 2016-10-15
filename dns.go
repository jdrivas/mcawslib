package mclib

import(
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/route53"
  "github.com/Sirupsen/logrus"


  // "awslib"
  "github.com/jdrivas/awslib"
)

// Returns the TLD that all proxies would be attached to.
// This can be used to filter DNS records.
// TODO: This needs to go in a configuration file.
func DefaultProxyTLD() (string) {
  return "hoods.momentlabs.io"
}

// Returns DNS records that are based on DefaultProxytTLD()
func GetDNSRecords(sess *session.Session) ([]*route53.ResourceRecordSet, error) {
  recordSet, err := awslib.ListDNSRecords(DefaultProxyTLD(), sess)
   return recordSet, err
}

// Returns DNS records that are based on p.DNSName()
func (p *Proxy) DNSRecords() ([]*route53.ResourceRecordSet, error) {
  recordSet, err := awslib.ListDNSRecords(p.DNSName(), p.AWSSession)
  return recordSet, err
}

// TODO: Much of this is probably best expressed as an interface.
// I also have to figure out the best way to remove all of the duplication betwen
// Server and Proxy. I haven't done that yet (though it's getting eggregious) because
// I'm not sure really how these relate. But ... something will have to be done soon.

//
// Proxy DNS
//

// This should probably be longer ......
const DefaultProxyTTL int64 = 60

// TODO: Clearly a work in progress. 
// TODO: Consider adding DNS Name and DNS Address to Proxy for caching purposes.
// Obviously this would require that we ensure that make sure that anything that comes
// though the proxy/server that updates DNS deals with the cache. It also means potentially
// reporing incorrect results in any number of cases where DNS is updated outside the
// scope of a single user and this library. Given that there is no performance issue
// to hand, let's leave it for now.

// This will do a DNS lookup for the expected name proxy: p.DNSName().
// If the record for that name does not have an entry for the 'execpted'
// ip address (p.PublicProxyIp) you will get an error. An mclib.DNSError is 
// returned with the name and the found addesses.
// In the future we might either let the function return the found values
// or the error object may have them.
// Other errors are possibile (e.g. record not found ....)
func (p *Proxy) PublicDNS() (fqdn, ipAddress string, err error) {

  records, err := p.DNSRecords()
  if err != nil { return fqdn, ipAddress, err }

  fqdn, ipAddress = "<unknown>", "<none>"
  expectedName := p.DNSName() + "." // the name in records return with the "." at the end.
  expectedIp := p.PublicProxyIp
  var nameR *route53.ResourceRecordSet
  var nameIp, foundIp string
  for _, r := range records {
    // There could be many records that have the same IP as the proxy.
    // that's what we do with forced hosts (ie. servers which we serve as proxy.)
    if *r.Name == expectedName {
      nameR = r 
      for _, rr := range r.ResourceRecords {
        if *rr.Value == expectedIp {
          nameIp = *rr.Value
          break
        }
        // Didin't find what we were looking for?
        if nameIp == "" && len(r.ResourceRecords) > 0 {
          foundIp = *r.ResourceRecords[0].Value
        }
      }
      break
    }
  }

  f := logrus.Fields{
    "foundDNSRecords": len(records),
    "proxy": p.Name,
    "publicIp": p.PublicProxyIp,
    "expectedName":  expectedName,
    "expectedIp": expectedIp,
    "foundName": fqdn,
    "foundIp": ipAddress,
  }
  switch {
  // Found nothing.
  case nameR == nil:
    err = NewDNSError("Failure to find a DNS record for this proxy", "<not-found>", []string{})
    log.Error(f, "Failed to get Proxy PublicDNS: No matching DNS.", err)
  // Found a record based on the name, but got something othesr than the expected ip.
  case foundIp != "":
    f["foundName"] = *nameR.Name
    f["foundIp"] = foundIp
    err = NewDNSErrorAWS("Unexpected IP address: found name but different ip", *nameR.Name, nameR.ResourceRecords)
    log.Error(f, "Failed to get Proxy PublicDNS: Unexpected IP address.", err)
  // Found a name but no ip. This really shouldn't happen in the normal course of things.
  case nameIp == "": 
    f["foundName"] = *nameR.Name
    err = NewDNSErrorAWS("No IP Address: found name but no IP", *nameR.Name, nameR.ResourceRecords)
    log.Error(f, "Failed to get Proxy PublicDNS: No IP address.", err)
  default:
    fqdn = *nameR.Name
    ipAddress = nameIp
    f["foundName"] = *nameR.Name
    f["foundIp"] = nameIp
    log.Info(f, "Retrieved Proxy DNS.")
  }
  return fqdn, ipAddress, err
}


// Returns a string that should be used as a DNS name for this server.
// This does not query DNS.
func (p *Proxy) DNSName() (dn string) {
  dn = nameToDNSForm(p.Name) + "." + DefaultProxyTLD()
  return dn
}

// Add/Update the DNS record for this proxy server.
// The DNS will create/update an A record for p.DomsinName() => p.PublicProxyIp()
// The expected IP address is the public IP address of the VM/Machine instance where the container is running.
func (p *Proxy) AttachToNetwork() (domainName string, changeInfo *route53.ChangeInfo, err error) {

  domainName = p.DNSName()
  comment := fmt.Sprintf("Attaching proxy: %s to network at: %s\n", domainName, p.PublicProxyIp)
  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, domainName, comment, DefaultProxyTTL ,p.AWSSession)

  return domainName, changeInfo, err
}

// Add/Update DNS for the server.
// The DNS will create/update an A record for s.DNSName() "." p.DNSName() => p.PublicProxyIp()
// The execpted IP address is the public IP of the VM/Machine instance where the proxy container is running.
func (p *Proxy) AttachToProxyNetwork(s *Server) (serverFQDN string, changeInfo *route53.ChangeInfo, err error) {

  serverFQDN = p.attachedServerFQDN(s)
  comment := fmt.Sprintf("Attaching server %s to network at: %s as %s\n", 
    s.Name, p.PublicProxyIp, serverFQDN)

  f := logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
    "serverFQDN": serverFQDN,
  }
  log.Info(f, "AttachToProxy: Updating Server DNS to point to proxy.")

  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, serverFQDN, comment, DefaultProxyTTL, p.AWSSession)
  return serverFQDN, changeInfo, err
}

// Remove DNS entry for Server pointing to this proxy.
// This looks for a DNS record as created by AttachToProxyNetwork and removes it.
// Fails if it can't find the DNS record.
func (p *Proxy) DetachFromProxyNetwork(s *Server) (changeInfo *route53.ChangeInfo, err error) {

  serverFQDN, err  := p.ProxiedServerFQDN(s)
  if err != nil {
    err = fmt.Errorf("Failed to obtain DNS for server: %s", err)
  } else {
    comment := fmt.Sprintf("Detaching server %s from proxy: Removing DNS record for: %s.",
      s.Name, serverFQDN)
    changeInfo, err = awslib.DetachFromDNS(p.PublicProxyIp, serverFQDN, comment, DefaultProxyTTL, p.AWSSession)
  }
  f := logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
    "serverFQDN": serverFQDN,
  }
  log.Info(f, "DetachFromProxy: Updating Server DNS to point to proxy.")
  return changeInfo, err
}

// Returns the server FQDN constructed from actual DNS for the proxy and the DNSName() from the server.
// Returns an error if it can't find the DNS for the proxy.
// serverFQDN does not have a  trailing ".".
// TODO. This should look up the actual servers record and behave like DNSName() above.
func (p* Proxy) ProxiedServerFQDN(s *Server) (serverFQDN string, err error) {
  proxyFQDN, _, err := p.PublicDNS()
  if err == nil {
    serverFQDN = fmt.Sprintf("%s.%s", s.DNSName(), proxyFQDN)
  }
  return strings.TrimSuffix(serverFQDN, "."), err
}

// This is what we expect to construct proxied server FQDN's out of.
func (p *Proxy) attachedServerFQDN(s *Server) (string) {
  return s.DNSName() + "." + p.DNSName()
}

//
// Server DNS
//

// Returns DNS records that are based on s.DomainName()
func (s *Server) DNSRecords() ([]*route53.ResourceRecordSet, error) {
  recordSet, err := awslib.ListDNSRecords(s.DNSName(), s.AWSSession)
  return recordSet, err
}

// TODO: This may all change as we move to shutting down public addresses for
// servers.

// This will do a DNS lookup for the expected name of the host: s.DNSName().
// If the record with the right name does not contain the expected address 
// (s.PublicServerIP) you will get an error. The error message will tell you what was found.
// In the future we might either let the function return the found values
// or the error object may have them.
// Other errors are possibile (e.g. record not found ....)
func (s *Server) PublicDNS() (fqdn, ipAddress string, err error) {
  records, err := s.DNSRecords()
  if err != nil { return fqdn, ipAddress, err }

  fqdn, ipAddress = "<unknown>", "<none>"
  expectedName := s.DNSName() + "." // the name in records return with the "." at the end.
  expectedIp := s.PublicServerIp

  var nameR *route53.ResourceRecordSet
  var nameIp, foundIp string
  for _, r := range records {
    // There could be many records that have the same IP as the server (especially when 
    // it's being managed by a proxy.)
    if *r.Name == expectedName {
      nameR = r 
      for _, rr := range r.ResourceRecords {
        if *rr.Value == expectedIp {
          nameIp = *rr.Value
          break
        }
        // Didin't find what we were looking for?
        if nameIp == "" && len(r.ResourceRecords) > 0 {
          foundIp = *r.ResourceRecords[0].Value
        }
      }
      break
    }
  }

  f := logrus.Fields{
    "foundDNSRecords": len(records),
    "server": s.Name,
    "serverIp": s.PublicServerIp,
    "expectedName":  expectedName,
    "expectedIp": expectedIp,
    "foundName": fqdn,
    "foundIp": ipAddress,
  }
  switch {
  // Found nothing.
  case nameR == nil:
    err = NewDNSError("Failure to find a DNS record for this server", fqdn, []string{})
    log.Error(f, "Failed to get Server PublicDNS: No matching DNS.", err)
  // Found a record based on the name, but got something other than the expected ip.
  case foundIp != "":
    f["foundName"] = *nameR.Name
    f["foundIp"] = foundIp
    err = NewDNSErrorAWS("Unexpected IP address: found name but different ip", *nameR.Name, nameR.ResourceRecords)
    log.Error(f, "Failed to get Server PublicDNS: Unexpected IP address.", err)
  // Found a name but no ip. This really shouldn't happen in the normal course of things.
  case nameIp == "": 
    f["foundName"] = *nameR.Name
    err = NewDNSErrorAWS("No IP Address: found name but no IP address", *nameR.Name, nameR.ResourceRecords)
    log.Error(f, "Failed to get Server PublicDNS: No IP address.", err)
  default:
    fqdn = *nameR.Name
    ipAddress = nameIp
    f["foundName"] = *nameR.Name
    f["foundIp"] = nameIp
    log.Info(f, "Retrieved Server DNS.")
  }
  return fqdn, ipAddress, err
}

// Name suitable for adding to a DNS address
func (s *Server) DNSName() (string) {
  return nameToDNSForm(s.Name)
}

//
// Shared Helpers
//

// Remove spaces and all lower case.
func nameToDNSForm(n string) (string) {
  return strings.ToLower(strings.Replace(n, " ", "-", -1))
}




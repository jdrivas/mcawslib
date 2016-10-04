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
func DefaultProxyTLD() (string) {
  return "hoods.momentlabs.io"
}

// 
// Get Published DNS
// Returns all the records that are based on the DefaultProxytTLD()
func GetDNSRecords(sess *session.Session) ([]*route53.ResourceRecordSet, error) {
  recordSet, err := awslib.ListDNSRecords(DefaultProxyTLD(), sess)
   return recordSet, err
}

// Returns DNS records attached to the proxy.
func (p *Proxy) DNSRecords() ([]*route53.ResourceRecordSet, error) {
  recordSet, err := awslib.ListDNSRecords(p.DomainName(), p.AWSSession)
  return recordSet, err
}

//
// Proxy DNS
//

// This should probably be longer ......
const DefaultProxyTTL int64 = 60

// TODO: This should do a DNS lookup to make sure ....
func (p *Proxy) PublicDNS() (string, string, error) {

  name, ipAddress := "<unknown>", "<none>"
  records, err := p.DNSRecords()
  if err != nil { return name, ipAddress, err }

  // There are a couple of things we need to check.
  // - The host the proxy is running on must have the public IP.
  // - we *expect* that the DNS name of the proxy will be derived
  // from the name of the proxy.
  dn := p.DomainName()
  ip := p.PublicProxyIp
  var nameR, ipR *route53.ResourceRecordSet
  nameIp, ipIp := "<none>", "<none>"
  for _, r := range records {
    if *r.Name == dn { 
      nameR = r
      if len(r.ResourceRecords) > 0 {
        nameIp = *r.ResourceRecords[0].Value
      }
    }
    for _, rr := range r.ResourceRecords {
      if *rr.Value == ip { 
        ipR = r 
        if len(r.ResourceRecords) > 0 {
          ipIp = *r.ResourceRecords[0].Value
        }
        break
      }
    }
    if nameR != nil && ipR != nil { break }
  }

  // If there is a descrepsnecy then send an error, but keep relevant values.
  // Currently this is an undocumented 'feature'. We could return an error object
  // withthe fou
  switch {
  case nameR == nil && ipR == nil:
    err = fmt.Errorf("Failure to find DNS record for either name (%s) or ip (%s)", dn, ip)
  case ipR == nil:
    name = *nameR.Name
    ipAddress = nameIp
    err = fmt.Errorf("UNEXPECTED IP: Failure to find record for expected ip (%s), " +
      "found record for expected name: (%s) record: (%s:%s)", ip, dn, name, ipAddress )
  case nameR  == nil:
    name = *ipR.Name
    ipAddress = ipIp
    err = fmt.Errorf("UNEXPECTED NAME: Failure to find record for expected name (%s), " + 
      "found record for expected ip: (%s) record: (%s:%s)", dn, ip, name, ipAddress)
  case nameR != ipR:
    err = fmt.Errorf("UNEXPECTED VALUES: Record for expected name (%s) " +
      "differed from record for expected IP (%s) name-record (%s:%s) ip-record(%s:%s)",
      dn, ip, *nameR.Name, nameIp, *ipR.Name, ipIp)
  default:
    name = *nameR.Name
    ipAddress = nameIp
  }

  return name, ipAddress, err
}

// Returns a string that should be used as a DNS name for this server.
func (p *Proxy) DomainName() (dn string) {
  dn = nameToDNSForm(p.Name) + "." + DefaultProxyTLD()
  return dn
}

// Add/Update the DNS record for this proxy server.
func (p *Proxy) AttachToNetwork() (domainName string, changeInfo *route53.ChangeInfo, err error) {

  domainName = p.DomainName()
  comment := fmt.Sprintf("Attaching proxy: %s to network at: %s\n", domainName, p.PublicProxyIp)
  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, domainName, comment, DefaultProxyTTL ,p.AWSSession)

  return domainName, changeInfo, err
}

// Add/Update DNS to add this server name off of the proxy network and assign the IP to the proxy.
func (p *Proxy) AttachToProxyNetwork(s *Server) (serverFQDN string, changeInfo *route53.ChangeInfo, err error) {

  // domainName := p.GetDomainName()
  // dnsName := s.DNSName()
  // serverFQDN = dnsName + "." + domainName
  serverFQDN = p.attachedServerFQDN(s)
  comment := fmt.Sprintf("Attaching server %s to network at: %s as %s\n", 
    s.Name, p.PublicProxyIp, serverFQDN)

  f:= logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
    "serverFQDN": serverFQDN,
  }
  log.Info(f, "Attaching server to proxy network.")

  // We do it this way, because we are using the session that is releant to 
  // the proxy. If somehow a different session was attached to the server
  // I don't think we'd want to use it. ...... Time will tell.
  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, serverFQDN, comment, DefaultProxyTTL, p.AWSSession)
  // changeInfo, err = s.AttachToNetwork(p.PublicProxyIp, serverFQDN, comment, p.AWSSession)
  return serverFQDN, changeInfo, err
}

func (p *Proxy) DetachFromProxyNetwork(s *Server) (changeInfo *route53.ChangeInfo, err error) {

  serverFQDN := p.attachedServerFQDN(s)
  comment := fmt.Sprintf("Detaching server %s from proxy: Removing DNS record for: %s.",
    s.Name, serverFQDN)
  changeInfo, err = awslib.DetachFromDNS(p.PublicProxyIp, serverFQDN, comment, DefaultProxyTTL, p.AWSSession)
  return changeInfo, err
}

func (p *Proxy) attachedServerFQDN(s *Server) (string) {
  return s.DNSName() + "." + p.DomainName()
}

//
// Server DNS
//


// Does lookups in DNS to find the DNS for this server.
// OR at least it should/will.
func (s *Server) DNSAddress() (string) {
  return s.PublicServerIp + ":" + s.ServerPort.String()
}

// Name suitable for adding to a DNS address
// Spaces removed and lowercased.
func (s *Server) DNSName() (string) {
  return nameToDNSForm(s.SafeServerName())
}

// Used on it's own in naming the server for a proxy.
// But also in DNS.
func (s *Server) SafeServerName() (string) {
  name := strings.Replace(s.Name, " ", "-", -1)
  return name
}

// Remove spaces and all lower case.
func nameToDNSForm(n string) (string) {
  return strings.ToLower(strings.Replace(n, " ", "-", -1))
}




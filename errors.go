package mclib

import (
  "fmt"
  "strings"

  "github.com/aws/aws-sdk-go/service/route53"

  "awslib"
  // "github.com/jdrivas/awslib"
)

type DNSError struct {
  Message string
  DNSName string
  IpAddresses []string
}

// Create a new error.
func NewDNSError(message, dnsName string, ipAddresses []string) (*DNSError) {
  return &DNSError{
    Message: message,
    DNSName: dnsName,
    IpAddresses: ipAddresses,
  }
}

// New error with points to ipAddresses 
func NewDNSerrorWP(message, dnsName string, ipAddresses []*string) (*DNSError) {
  return NewDNSError(message, dnsName, awslib.StringSlice(ipAddresses))
}

// New error with from route53.ResourceRecords
func NewDNSErrorAWS(message, dnsName string, resources []*route53.ResourceRecord) (*DNSError) {
  ips := make([]string, len(resources))
  for i, v := range resources {
    ips[i] = *v.Value
  }
  return NewDNSError(message, dnsName, ips)
}

func (e DNSError) Error() string {
  return fmt.Sprintf("%s: %s(%s)", e.Message, e.DNSName, strings.Join(e.IpAddresses, ","))
}
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pavel-genai/dns-resolver/pkg/dns"
)

func main() {
	recordType := flag.String("type", "A", "DNS record type (A, AAAA, CNAME, MX, NS, TXT)")
	verbose := flag.Bool("verbose", false, "Enable verbose output showing resolution steps")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <domain>\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	domain := flag.Arg(0)
	qtype, err := dns.StringToType(*recordType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resolver := dns.NewResolver(*verbose)

	records, err := resolver.Resolve(domain, qtype)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Printf("No %s records found for %s\n", strings.ToUpper(*recordType), domain)
		os.Exit(0)
	}

	for _, rr := range records {
		fmt.Printf("%s\t%d\tIN\t%s\t%s\n", rr.Name, rr.TTL, dns.TypeToString(rr.Type), rr.ParsedData)
	}
}

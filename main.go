package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	dns "google.golang.org/api/dns/v1"
)

var (
	address = flag.String("address", "", "Address to set (required or -interface or -query)")
	ifName  = flag.String("interface", "", "Interface to get first address from (required or -address or -query)")
	query   = flag.Bool("query", false, "Query public IP address (required or -interface or -address)")
	host    = flag.String("host", "", "Host to update (required)")
	project = flag.String("project", "", "GCP project (required)")
	zoneID  = flag.String("zone", "", "GCP Zone ID (required)")
)

func usageExit(err string) {
	fmt.Println(err)
	fmt.Println()
	flag.Usage()
	os.Exit(1)
}

func main() {
	flag.Parse()

	if (*address == "" && *ifName == "" && !*query) || *host == "" || *project == "" || *zoneID == "" {
		usageExit("(-address or -interface or -query), -host, -project and -zone are required")
	}
	if (*address != "" && (*ifName != "" || *query)) || (*ifName != "" && *query) {
		usageExit("-address, -interface and -query are mutually exclusive")
	}

	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", *host), 0)

	var wantedAddress string
	switch {
	case *address != "":
		wantedAddress = *address

	case *ifName != "":
		iface, err := net.InterfaceByName(*ifName)
		if err != nil {
			logger.Fatalln("Couldn't get interface:", err)
		}
		ifAddrs, err := iface.Addrs()
		if err != nil {
			logger.Fatalln("Couldn't get interface addrs:", err)
		}
		if len(ifAddrs) == 0 {
			logger.Fatalln("No interface addresses")
		}
		ifaceIP, _, err := net.ParseCIDR(ifAddrs[0].String())
		if err != nil {
			logger.Fatalln("Couldn't parse interface address:", ifAddrs[0].String())
		}
		wantedAddress = ifaceIP.String()

	case *query:
		res, err := http.Get("https://api.ipify.org")
		if err != nil {
			logger.Fatalln("Failed to query api.ipify.org:", err)
		}
		if res.StatusCode > 299 {
			logger.Fatalln("Weird status code from api.ipify.org:", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Fatalln("Failed to read response body:", err)
		}
		wantedAddress = strings.TrimSpace(string(body))
	}

	if wantedAddress == "" {
		logger.Fatalln("Empty address returned, aborting")
	}
	ip := net.ParseIP(wantedAddress)
	if ip == nil {
		logger.Fatalln("Couldn't parse address as valid IP:", wantedAddress)
	}
	logger.Println("Using address:", wantedAddress)

	ctx := context.Background()

	dnsService, err := dns.NewService(ctx)
	if err != nil {
		logger.Fatalln("Failed to create DNS service:", err)
	}

	errDone := fmt.Errorf("done")
	var record *dns.ResourceRecordSet
	err = dnsService.ResourceRecordSets.List(*project, *zoneID).Pages(
		ctx,
		func(page *dns.ResourceRecordSetsListResponse) error {
			for _, r := range page.Rrsets {
				if r.Name == *host && r.Type == "A" {
					record = r
					return errDone
				}
			}
			return nil
		},
	)
	if err != nil && !errors.Is(err, errDone) {
		logger.Fatalln("Error listing ResourceRecordSets:", err)
	}
	if record == nil {
		logger.Fatalln("No record found")
	}

	if len(record.Rrdatas) > 1 {
		logger.Fatalln("Unexpected amount of record values:", record.Rrdatas)
	}

	currentAddress := record.Rrdatas[0]
	if wantedAddress == currentAddress {
		logger.Println("No change needed:", wantedAddress)
		return
	}

	logger.Printf("Updating %s to %s...", *host, wantedAddress)
	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{{
			Name:    *host,
			Type:    "A",
			Ttl:     record.Ttl,
			Rrdatas: []string{wantedAddress},
		}},
		Deletions: []*dns.ResourceRecordSet{record},
	}

	_, err = dnsService.Changes.Create(*project, *zoneID, change).Context(ctx).Do()
	if err != nil {
		logger.Fatalln("Failed to update record:", err)
	}
	logger.Println("Updated.")
}

package main

import (
    "flag"
    "fmt"
    "regexp"

    "io/ioutil"
    "log"
    "net/http"

    "github.com/cloudflare/cloudflare-go"
)

const PUBLIC_IP_URL = "https://api.ipify.org";

var regExpIPv4 = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`);

func main() {
    cfApiKey, cfApiEmail, cfApiZone, cfApiDomain, publicIpToFilename := getFlags();

    if cfApiKey == "" || cfApiEmail == "" || cfApiZone == "" || cfApiDomain == "" || publicIpToFilename == "" {
        println("Please provide all params for the script:");
        flag.PrintDefaults();
        log.Fatal();
    }

    publicIp := getPublicIp();
    fmt.Printf("Public IP is %s\n", publicIp);
    currentIp := readFileContent(publicIpToFilename);
    fmt.Printf("Current IP is %s\n", currentIp);

    if publicIp == currentIp {
        return;
    }

    api, err := cloudflare.New(cfApiKey, cfApiEmail);
    if err != nil {
        log.Fatal(err)
    }
    

    // Fetch the zone ID
    zoneId, err := api.ZoneIDByName(cfApiZone)
    if err != nil {
        log.Fatal(err)
    }

    // Fetch all records for a zone
    dnsRecords, err := api.DNSRecords(zoneId, cloudflare.DNSRecord{ Name: cfApiDomain })
    if err != nil {
        log.Fatal(err)
    }

    var dnsRecord cloudflare.DNSRecord;
    for _, record := range dnsRecords {
        if record.Name == cfApiDomain {
            dnsRecord = record;
        }
    }

    if dnsRecord.Name != cfApiDomain {
        log.Fatal("Cant find domain " + cfApiDomain)
    }

    fmt.Printf("%s: %s\n", cfApiDomain, publicIp);

    if dnsRecord.Content != publicIp {
        fmt.Printf("Updating IP for %s\n", cfApiDomain);

        dnsRecord.Content = publicIp;
        err := api.UpdateDNSRecord(zoneId, dnsRecord.ID, dnsRecord);
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("IP is updated\n");
    }

    saveFileContent(publicIpToFilename, publicIp);

    fmt.Printf("Done\n");
}

func getFlags() (string, string, string, string, string) {
    cfApiKeyPointer := flag.String("cf-api-key", "", "Please provide CF API KEY");
    cfApiEmailPointer := flag.String("cf-api-email", "", "Please provide CF API EMAIL");
    cfApiZonePointer := flag.String("cf-api-zone", "", "Please provide CF API ZONE");
    cfApiDomainPointer := flag.String("cf-api-domain", "", "Please provide CF API DOMAIN");
    publicIpToFilenamePointer := flag.String("public-ip-filename", "", "Please filename with public ip");
    flag.Parse();

    return *cfApiKeyPointer, *cfApiEmailPointer, *cfApiZonePointer, *cfApiDomainPointer, *publicIpToFilenamePointer;
}

func readFileContent(filename string) string {
    content, err := ioutil.ReadFile(filename)
    if err != nil {
        return "";
    }
    return string(content);
}

func saveFileContent(filename, data string)  {
    err := ioutil.WriteFile(filename, []byte(data), 0644)
    if err != nil {
        log.Fatal(err)
    }
}

func getPublicIp() string {
    resp, err := http.Get(PUBLIC_IP_URL);
    if err != nil {
        log.Fatal(err)
    }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatal(err)
    }
    ip := string(body);
    if !isIPv4(ip) {
        log.Fatal("Cant get IPv4 from " + PUBLIC_IP_URL)
    }
    return ip;
}

func isIPv4(ip string) bool {
    if ip == "" {
        return false;
    }
    return regExpIPv4.MatchString(ip);
}
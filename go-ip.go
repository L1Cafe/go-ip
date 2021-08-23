package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
)

func isIPv4(ip string) bool {
	// Because net/http returns the IP with the port number, any string containing
	// the ":" more than twice is guaranteed to be an IPv6, as IPv4 has it only
	// once, and IPv6 has it at least twice for the host part, plus once again
	// for the port
	return strings.Count(ip, ":") < 2
}

func cleanUpIpv4(ip string) (string, error) {
	// TODO is this idiomatic?
	var result string
	err := errors.New("not an IPv4 address")

	result_arr := strings.Split(ip, ":")
	if net.ParseIP(result_arr[0]) != nil {
		result = result_arr[0]
		err = nil
	}
	return result, err
}

func cleanUpIpv6(ip string) (string, error) {
	// TODO is this idiomatic?
	var result string
	err := errors.New("not an IPv6 address")

	if strings.Contains(ip, "[") && !strings.Contains(ip, ".") {
		ipv6_regex := regexp.MustCompile(`\[(.*)\]`)
		ipv6_match := ipv6_regex.FindAllStringSubmatch(ip, 1)
		if net.ParseIP(ipv6_match[0][1]) != nil {
			result = string(ipv6_match[0][1])
			err = nil
		}
	}
	return result, err
}

func getIpFromRequest(r *http.Request) (string, error) {
	var remote_host string
	var err error

	if isIPv4(r.RemoteAddr) {
		remote_host, err = cleanUpIpv4(r.RemoteAddr)
	} else {
		remote_host, err = cleanUpIpv6(r.RemoteAddr)
	}

	return remote_host, err
}

func returnFullInfo(w http.ResponseWriter, r *http.Request) {
	var headers string

	for k, v := range r.Header {
		headers = headers + strings.ToUpper(k)
		headers = headers + " : "
		for _, s := range v {
			headers = headers + s + " "
		}
		headers = headers + "\n"
	}
	io.WriteString(w, headers)
	ip, err := getIpFromRequest(r)
	if err == nil {
		io.WriteString(w, "------------------------------------\n")
		io.WriteString(w, "IP: ")
		io.WriteString(w, ip)
		io.WriteString(w, "\n")
	} else {
		io.WriteString(w, err.Error())
	}
}

func returnIp(w http.ResponseWriter, r *http.Request) {
	// TODO: need to check for other ways of getting the IP
	if r.URL.Path != "/" {
		// redirect just in case
		// may add functionality later, that's why it's a temporary redirect
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else {
		remote_host, err := getIpFromRequest(r)
		if err == nil {
			io.WriteString(w, remote_host)
		} else {
			io.WriteString(w, err.Error())
		}
	}
}

func main() {
	http.HandleFunc("/", returnIp)
	http.HandleFunc("/full", returnFullInfo)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

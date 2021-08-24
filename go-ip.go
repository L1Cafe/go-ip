package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

func isIPv4(ip string) bool {
	// Because net/http returns the IP with the port number, any string containing
	// the ":" more than twice is guaranteed to be an IPv6, as IPv4 has it only
	// once, and IPv6 has it at least twice for the host part, plus once again
	// for the port
	return strings.Count(ip, ":") < 2
}

// cleanUpIpv4 takes a string of the format 192.168.113.1:884245 and converts it
// into 192.168.113.1
func cleanUpIpv4(ip string) (string, error) {
	// TODO is this idiomatic?
	var result string
	err := errors.New("error parsing IPv4 address")

	result_arr := strings.Split(ip, ":")
	if net.ParseIP(result_arr[0]) != nil {
		result = result_arr[0]
		err = nil
	}
	return result, err
}

// cleanUpIpv6 takes a string of the format [::1]:12354 and converts it into ::1
func cleanUpIpv6(ip string) (string, error) {
	// TODO is this idiomatic?
	var result string
	err := errors.New("error parsing IPv6 address")

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

// getIpFromRequest will only take the IP address in the TCP packets, instead
// of trying to detect headers.
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

// getAnyFromRequest gets the first available IP address by trying different
// methods. If it can't find the IP by these means, it falls back to detecting
// remote host IP.
func getAnyFromRequest(r *http.Request) (string, error) {
	var ip string
	var err error
	try_headers := []string{"X-Forwarded-For", "X-Real-Ip", "X-True-Client-Ip",
		"True-Client-Ip", "X-Originating-Ip", "X-Remote-Ip", "X-Remote-Addr"}
	// Forwarded (??)
	headers := r.Header

	for _, header := range try_headers {
		ip = headers.Get(header)
		if ip != "" {
			break
		}
	}

	if net.ParseIP(ip) == nil {
		ip, err = getIpFromRequest(r)
	}
	return ip, err
}

// returnIp is the most basic functionality for this application, located at
// the root directory
func returnIp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// redirect just in case
		// may add functionality later, that's why it's a temporary redirect
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else {
		remote_host, err := getAnyFromRequest(r)
		if err == nil {
			io.WriteString(w, remote_host)
		} else {
			io.WriteString(w, err.Error())
		}
	}
}

// returnFullInfo returns all headers from the browser, and finally returns
// the IP address
func returnFullInfo(w http.ResponseWriter, r *http.Request) {
	var header_keys []string
	var headers string

	for k := range r.Header {
		header_keys = append(header_keys, strings.ToUpper(k))
	}

	sort.Strings(header_keys)

	for _, k := range header_keys {
		headers = headers + k + " : " + r.Header.Get(k) + "\n"
	}

	io.WriteString(w, headers)
	ip, err := getIpFromRequest(r)
	if err == nil {
		io.WriteString(w, "------------------------------------\n")
		io.WriteString(w, "Source IP : ")
		io.WriteString(w, ip)
		io.WriteString(w, "\n")
	} else {
		io.WriteString(w, err.Error())
	}
}

// returnSourceIp ignores all headers and literally just returns the TCP/IP
// packet source address
func returnSourceIp(w http.ResponseWriter, r *http.Request) {
	remote_host, err := getIpFromRequest(r)
	if err == nil {
		io.WriteString(w, remote_host)
	} else {
		io.WriteString(w, err.Error())
	}
}

func main() {
	http.HandleFunc("/", returnIp)
	http.HandleFunc("/full", returnFullInfo)
	http.HandleFunc("/source-ip", returnSourceIp) // forces return of source IP
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

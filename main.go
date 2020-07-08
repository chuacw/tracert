package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"log"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// c.IPv4PacketConn().SetTTL(64) // for ipv4
// c.IPv6PacketConn().HopLimit(64) // for ipv6

func pingIPv4(ttl int, ip net.IP) (*icmp.Message, string, error) {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("listen err, %s", err)
	}
	c.IPv4PacketConn().SetTTL(ttl)
	defer c.Close()

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := c.WriteTo(wb, &net.IPAddr{IP: ip}); err != nil {
		log.Fatalf("WriteTo err, %s", err)
	}
	c.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	rb := make([]byte, 1500)
	n, fromIP, err := c.ReadFrom(rb)
	if err != nil {
		log.Fatal(err)
	}
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), rb[:n])
	return rm, fromIP.String(), err
}

func pingIPv6(ttl int, ip net.IP) (*icmp.Message, string, error) {
	c, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		log.Fatalf("listen err, %s", err)
	}
	defer c.Close()

	c.IPv6PacketConn().SetHopLimit(ttl)
	c.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)

	wm := icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := c.WriteTo(wb, &net.IPAddr{IP: ip}); err != nil {
		log.Fatalf("WriteTo err, %s", err)
	}
	c.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	rb := make([]byte, 1500)
	n, fromIP, err := c.ReadFrom(rb)
	var e *net.OpError
	if errors.As(err, &e) {
		rm := &icmp.Message{Type: ipv6.ICMPTypeTimeExceeded}
		return rm, "*", nil
	}
	rm, err := icmp.ParseMessage(ipv6.ICMPTypeEchoRequest.Protocol(), rb[:n])
	return rm, fromIP.String(), err
}

func icmpping(ttl int, targetAddrOrIP string) (*icmp.Message, string, error) {

	targetIPs, err := net.LookupIP(targetAddrOrIP)
	var targetIP net.IP
	for _, answer := range targetIPs {
		if answer.To4() != nil {
			targetIP = answer.To4()
			break
		}
	}
	rm, fromIP, err := pingIPv4(ttl, targetIP)
	return rm, fromIP, err
}

func UNUSED(i interface{}) {}

var forceIPv4, forceIPv6 bool

func traceroute(targetAddrOrIP string) {

	targetIPs, err := net.LookupIP(targetAddrOrIP)
	var (
		targetIP net.IP
		resolved bool = false
		funcping func(ttl int, ip net.IP) (*icmp.Message, string, error)
	)
	if forceIPv6 {
		funcping = pingIPv6
		for _, answer := range targetIPs {
			if answer.To16() != nil {
				targetIP = answer.To16()
				resolved = true
				break
			}
		}
	} else if forceIPv4 {
		funcping = pingIPv4
		for _, answer := range targetIPs {
			if answer.To4() != nil {
				targetIP = answer.To4()
				resolved = true
				break
			}
		}
	}
	if !resolved {
		fmt.Printf("Unable to resolve %s.\n", targetAddrOrIP)
		os.Exit(1)
	}
	if targetAddrOrIP != targetIP.String() {
		fmt.Printf("Resolved %s as %s.\n\n", targetAddrOrIP, targetIP)
	}
	UNUSED(err)

	var ttl int = 1
	for {
		start := time.Now()
		rm, fromIP, err := funcping(ttl, targetIP)
		UNUSED(rm)
		UNUSED(fromIP)
		if err != nil {

		}
		elapsed := time.Since(start)
		UNUSED(elapsed)
		var sElapsed string
		if fromIP == "*" {
			sElapsed = "Request timed out."
		} else {
			sElapsed = fmt.Sprintf("%s", elapsed)
		}
		fmt.Printf("%d\t%s,\telapsed: %s\n", ttl, fromIP, sElapsed)
		if rm.Type == ipv4.ICMPTypeTimeExceeded || rm.Type == ipv6.ICMPTypeTimeExceeded {
			ttl++
			continue
		}
		break
	}
}

func parseCmdLine() {
	flag.BoolVar(&forceIPv4, "4", true, "Force using IPv4.")
	flag.BoolVar(&forceIPv6, "6", false, "Force using IPv6.")
	flag.Parse()
	if forceIPv6 {
		forceIPv4 = false
	}
}

// working prototype, tested in VS Code, 29 Jun 2020
func main() {
	fmt.Println("traceroute v0.1 (c) 2020 chuacw")
	parseCmdLine()
	cmds := flag.Args()
	var targetIP string
	if len(cmds) > 0 {
		targetIP = cmds[0]
	}
	traceroute(targetIP)
}

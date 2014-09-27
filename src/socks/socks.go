// Copyright 2012, Hailiang Wang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package socks implements a SOCKS (SOCKS4, SOCKS4A and SOCKS5) proxy client.

A complete example using this package:
  package main

  import (
    "github.com/hailiang/socks"
    "fmt"
    "net/http"
    "io/ioutil"
  )

  func main() {
    dialSocksProxy := socks.DialSocksProxy(socks.SOCKS5, "127.0.0.1:1080")
    tr := &http.Transport{Dial: dialSocksProxy}
    httpClient := &http.Client{Transport: tr}

    bodyText, err := TestHttpsGet(httpClient, "https://github.com/about")
    if err != nil {
      fmt.Println(err.Error())
    }
    fmt.Print(bodyText)
  }

  func TestHttpsGet(c *http.Client, url string) (bodyText string, err error) {
    resp, err := c.Get(url)
    if err != nil { return }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil { return }
    bodyText = string(body)
    return
  }
*/
package socks

import (
  "io"
  "fmt"
  "net"
  "errors"
  "strconv"
)

// Argument socksType should be one of SOCKS4, SOCKS4A and SOCKS5.
// Argument proxy should be in this format "127.0.0.1:1080".
func DialSocksProxy(proxy string, dialer net.Dialer) func(string, string) (net.Conn, error) {
  return func(_, targetAddr string) (conn net.Conn, err error) {
    return dialSocks5(proxy, targetAddr, dialer)
  }
}

func dialSocks5(proxy, targetAddr string, dialer net.Dialer) (conn net.Conn, err error) {
  
  // dial TCP
  conn, err = dialer.Dial("tcp", proxy)
  if err != nil {
    return
  }

  // version identifier/method selection request
  req := []byte{
    5, // version number
    1, // number of methods
    0, // method 0: no authentication (only anonymous access supported for now)
  }
  resp, err := sendReceive(conn, req, 2)
  if err != nil {
    return
  } else if resp[0] != 5 {
    err = errors.New("Server does not support Socks 5.")
  } else if resp[1] != 0 { // no auth
    err = errors.New("socks method negotiation failed.")
    return
  }

  // detail request
  host, port, err := splitHostPort(targetAddr)
  req = []byte{
    5,               // version number
    1,               // connect command
    0,               // reserved, must be zero
    3,               // address type, 3 means domain name
    byte(len(host)), // address length
  }
  req = append(req, []byte(host)...)
  req = append(req, []byte{
    byte(port >> 8), // higher byte of destination port
    byte(port),      // lower byte of destination port (big endian)
  }...)
  resp, err = sendReceive(conn, req, 10)
  if err != nil {
    return
  } else if resp[1] != 0 {
    err = errors.New("Can't complete SOCKS5 connection.")
  }

  return
}

func sendReceive(conn net.Conn, req []byte, expect int) (resp []byte, err error) {
  _, err = conn.Write(req)
  if err != nil {
    return
  }
  resp, err = readAll(conn, expect)
  return
}

func readAll(conn net.Conn, expect int) (resp []byte, err error) {
  resp = make([]byte, expect)
  n, err := io.ReadFull(conn, resp)
  resp = resp[:n]
  return
}

func lookupIP(host string) (ip net.IP, err error) {
  ips, err := net.LookupIP(host)
  if err != nil {
    return
  }
  if len(ips) == 0 {
    err = errors.New(fmt.Sprintf("Cannot resolve host: %s.", host))
    return
  }
  ip = ips[0].To4()
  if len(ip) != net.IPv4len {
    fmt.Println(len(ip), ip)
    err = errors.New("IPv6 is not supported by SOCKS4.")
    return
  }
  return
}

func splitHostPort(addr string) (host string, port uint16, err error) {
  host, portStr, err := net.SplitHostPort(addr)
  portInt, err := strconv.ParseUint(portStr, 10, 16)
  port = uint16(portInt)
  return
}

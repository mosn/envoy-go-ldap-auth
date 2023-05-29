package main

import (
	"envoy-go-ldap-auth/example"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
)

func init() {
	http.RegisterHttpFilterConfigFactory("envoy-go-ldap-auth", example.ConfigFactory)
	http.RegisterHttpFilterConfigParser(&example.Parser{})
}

func main() {

}

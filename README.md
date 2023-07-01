envoy-go-ldap-auth
==================

This is a simple LDAP auth filter for envoy written in go. Only requests that pass the LDAP server's authentication will be proxied to the upstream service.

## Status

This is under active development and is not ready for production use.

## Usage

The client set credentials in `Authorization` header in the following format:

```Plaintext
credentials := Basic base64(username:password)
```

An example of the `Authorization` header is as follows (`aGFja2Vyczpkb2dvb2Q=`, which is the base64-encoded value of `hackers:dogood`):

```Plaintext
Authorization: Basic aGFja2Vyczpkb2dvb2Q=
```

Configure your envoy.yaml, include required fields: host, port, baseDn and attribute.

```yaml
http_filters:
  - name: envoy.filters.http.golang
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
      library_id: example
      library_path: /etc/envoy/libgolang.so
      plugin_name: envoy-go-ldap-auth
      plugin_config:
        "@type": type.googleapis.com/xds.type.v3.TypedStruct
        value:
          # required
          host: localhost
          port: 3894
          baseDn: dc=glauth,dc=com
          attribute: cn
          # optional
          # be used in search mode
          bindDn: # cn=admin,dc=example,dc=com
          bindPassword: # mypassword
          # if the filter is set, the filter application will run in search mode.
          filter: # (&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%s))
          timeout: 60 # unit is second.
          tls: # false
          startTLS: # false
          insecureSkipVerify: # false
          rootCA: # ""
```

Then, you can start your filter.

```bash
make build
make run 
```

## Test

This test case is based on glauth and can be utilized to evaluate your filter.

Firstly, download [glauth](https://github.com/glauth/glauth/releases), and its [sample config file](https://github.com/glauth/glauth/blob/master/v2/sample-simple.cfg).

```bash
# download glauth
curl -L -o glauth https://github.com/glauth/glauth/releases/download/v2.2.0-RC1/glauth-linux-amd64
chmod +x glauth

# download sample config file of glauth
curl -L -o sample.cfg https://raw.githubusercontent.com/glauth/glauth/master/v2/sample-simple.cfg
```

Then, start glauth.

```bash
./glauth -c sample.cfg
```

Run it with the example config file.

```bash
go test test/e2e_bind_test.go test/common.go
```

## Bind Mode and Search Mode

If no filter is specified in its configuration, the middleware runs in the default bind mode, meaning it tries to make a simple bind request to the LDAP server with the credentials provided in the request headers. If the bind succeeds, the middleware forwards the request, otherwise it returns a `401 Unauthorized` status code.

If a filter query is specified in the middleware configuration, and the Authentication Source referenced has a `bindDN` and a `bindPassword`, then the middleware runs in search mode. In this mode, a search query with the given filter is issued to the LDAP server before trying to bind. If result of this search returns only 1 record, it tries to issue a bind request with this record, otherwise it aborts a `401 Unauthorized` status code.

## Configurations

### Required

- host, string, default "localhost", required

Host on which the LDAP server is running.

- port, number, default 389, required

TCP port where the LDAP server is listening. 389 is the default port for LDAP.

- baseDn, string, "dc=example,dc=com", required

The `baseDN` option should be set to the base domain name that should be used for bind and search queries.

- attribute, string, default "cn", required

Attribute to be used to search the user; e.g., “cn”.

### Optional

- filter, string, default ""

If not empty, the middleware will run in search mode, filtering search results with the given query.

Filter queries can use the `%s` placeholder that is replaced by the username provided in the `Authorization` header of the request. For example: `(&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%s))`, `(cn=%s)`.

- bindDn, string, default ""

The domain name to bind to in order to authenticate to the LDAP server when running on search mode. Leaving this empty with search mode means binds are anonymous, which is rarely expected behavior. It is not used when running in bind_mode.

- bindPassword, string, default ""

The password corresponding to the `bindDN` specified when running in search mode, used in order to authenticate to the LDAP server.

- timeout, number, default 60

An optional timeout in seconds when waiting for connection with LDAP server.

- tls, bool, default false

Set to true if LDAP server should use an encrypted TLS connection, either with StartTLS or regular TLS.

- startTLS, bool, default false

If set to true, instructs this filter to issue a StartTLS request(sends the command to start a TLS session and then creates a new TLS Client) when initializing the connection with the LDAP server. If the startTLS setting is enabled, it is important to ensure that the tls setting is also enabled.

- insecureSkipVerify, bool, default false

When TLS is enabled, the connection to the LDAP server is verified to be secure. This option allows the filter to proceed and operate even for server connections otherwise considered insecure.

- rootCA, string, default ""

The rootCA option should contain one or more PEM-encoded certificates to use to establish a connection with the LDAP server if the connection uses TLS but that the certificate was signed by a custom Certificate Authority.
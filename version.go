package hawksdk

// Version of the hawk-sdk-go library. Used in the User-Agent header on
// outbound HTTP requests so a misbehaving SDK can be identified by daemon
// logs and operators.
const Version = "0.2.0"

// userAgent returns the User-Agent string for outbound HTTP requests.
func userAgent() string { return "hawk-sdk-go/" + Version }

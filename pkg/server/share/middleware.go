package share

import (
	"crypto/x509"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryo-arima/cmn-weave/pkg/global"
)

const peerCertKey = "mtls_peer_cert"

// PeerFromGinContext returns the verified peer certificate stored by
// RequireMTLS. The boolean return is false when the middleware was not
// applied or the request carried no client certificate.
func PeerFromGinContext(c *gin.Context) (*x509.Certificate, bool) {
	val, ok := c.Get(peerCertKey)
	if !ok {
		return nil, false
	}
	cert, ok := val.(*x509.Certificate)
	return cert, ok
}

// RequireMTLS is a Gin middleware that rejects requests without a verified
// client certificate and stores the leaf certificate in the Gin context
// under peerCertKey for downstream handlers.
//
// The TLS listener must be configured with tls.RequireAndVerifyClientCert;
// this middleware is a defence-in-depth check that also exposes the peer
// identity to controllers and access logs.
func RequireMTLS() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GetRequestID(c)
		if c.Request.TLS == nil || len(c.Request.TLS.PeerCertificates) == 0 {
			GetServerLogger().WARN(requestID, global.SMMTLS3,
				c.Request.RemoteAddr+" "+c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "client certificate required"},
			)
			return
		}
		c.Set(peerCertKey, c.Request.TLS.PeerCertificates[0])
		GetServerLogger().INFO(requestID, global.SMMTLS2,
			c.Request.TLS.PeerCertificates[0].Subject.CommonName,
		)
		c.Next()
	}
}

package handler

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/pki-service/internal/crypto"
	"github.com/helixdevelopment/pki-service/internal/model"
	"github.com/helixdevelopment/pki-service/internal/repository"
)

// Handler holds PKI service handlers.
type Handler struct {
	repo   repository.Repository
	encKey string
}

// New returns a new Handler with dependencies.
func New(repo repository.Repository, encKey string) *Handler {
	return &Handler{repo: repo, encKey: encKey}
}

// CreateCA handles CA creation.
func (h *Handler) CreateCA(c *gin.Context) {
	var req model.CreateCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	privPEM, pubPEM, err := crypto.GenerateCAKeyPair(2048)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate CA key pair"})
		return
	}

	subject := "CN=" + req.Name + " CA,O=Helix,C=US"
	// The second return value is the CA's OWN self-signed X.509 serial
	// number (a random 128-bit value, correctly embedded in certPEM
	// already) — it is intentionally NOT stored in
	// CertificateAuthority.SerialNumber. That column is a DIFFERENT
	// thing: the monotonic counter GetNextSerialNumber increments to
	// assign serials to CHILD certificates issued under this CA. It
	// MUST start at 0 (see below) and MUST NEVER be seeded from a
	// random 128-bit value truncated via big.Int.Int64() — that
	// truncation is undefined/effectively-random-signed (Go big.Int
	// docs: "If x cannot be represented in an int64, the result is
	// undefined") and was ~50% likely to start the per-CA counter
	// negative, which then poisoned every certificate issued under
	// that CA with a negative serial and made x509.CreateCertificate
	// reject it with "x509: serial number must be positive" — a real
	// defect discovered ONLY via the real-persistence, real-x509-signing
	// integration test (queue#4, §11.4.27); no unit test with a mocked
	// repository could ever have caught it.
	certPEM, _, err := crypto.CreateCACertificate(privPEM, subject, req.ValidityDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create CA certificate"})
		return
	}

	encPriv, err := crypto.EncryptPrivateKey(privPEM, h.encKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt CA private key"})
		return
	}

	now := time.Now().UTC()
	ca := &model.CertificateAuthority{
		ID:           uuid.New(),
		OrgID:        orgID,
		Name:         req.Name,
		Description:  req.Description,
		CACertPEM:    certPEM,
		CAKeyPEM:     encPriv,
		SerialNumber: 0, // child-certificate serial counter — starts at 0, see comment above
		ValidityDays: req.ValidityDays,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.CreateCA(c.Request.Context(), ca); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store CA"})
		return
	}

	_ = pubPEM
	c.JSON(http.StatusCreated, model.CAResponse{
		ID:           ca.ID,
		OrgID:        ca.OrgID,
		Name:         ca.Name,
		Description:  ca.Description,
		CACertPEM:    ca.CACertPEM,
		SerialNumber: ca.SerialNumber,
		ValidityDays: ca.ValidityDays,
		CreatedAt:    ca.CreatedAt,
		UpdatedAt:    ca.UpdatedAt,
	})
}

// ListCAs handles listing CAs.
func (h *Handler) ListCAs(c *gin.Context) {
	orgIDStr := c.Query("org_id")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id is required"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := uuid.Parse(l); err == nil {
			// ignore
		}
	}
	_ = c.Query("offset")

	cas, err := h.repo.ListCAs(c.Request.Context(), orgID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list CAs"})
		return
	}

	var resp []model.CAResponse
	for _, ca := range cas {
		resp = append(resp, model.CAResponse{
			ID:           ca.ID,
			OrgID:        ca.OrgID,
			Name:         ca.Name,
			Description:  ca.Description,
			CACertPEM:    ca.CACertPEM,
			SerialNumber: ca.SerialNumber,
			ValidityDays: ca.ValidityDays,
			CreatedAt:    ca.CreatedAt,
			UpdatedAt:    ca.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"cas": resp, "limit": limit, "offset": offset})
}

// GetCA handles retrieving a single CA.
func (h *Handler) GetCA(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ca, err := h.repo.GetCAByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA not found"})
		return
	}

	c.JSON(http.StatusOK, model.CAResponse{
		ID:           ca.ID,
		OrgID:        ca.OrgID,
		Name:         ca.Name,
		Description:  ca.Description,
		CACertPEM:    ca.CACertPEM,
		SerialNumber: ca.SerialNumber,
		ValidityDays: ca.ValidityDays,
		CreatedAt:    ca.CreatedAt,
		UpdatedAt:    ca.UpdatedAt,
	})
}

// DeleteCA handles CA deletion.
func (h *Handler) DeleteCA(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteCA(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete CA"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateCertificate handles certificate creation under a CA.
func (h *Handler) CreateCertificate(c *gin.Context) {
	caID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ca_id"})
		return
	}

	var req model.CreateCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ca, err := h.repo.GetCAByID(c.Request.Context(), caID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA not found"})
		return
	}

	caPrivPEM, err := crypto.DecryptPrivateKey(ca.CAKeyPEM, h.encKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt CA private key"})
		return
	}

	privPEM, pubPEM, err := crypto.GenerateCertKeyPair(2048)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate certificate key pair"})
		return
	}

	serial, err := h.repo.GetNextSerialNumber(c.Request.Context(), caID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get next serial number"})
		return
	}

	certPEM, err := crypto.CreateCertificate(privPEM, caPrivPEM, ca.CACertPEM, req.Subject, big.NewInt(serial), req.ValidityDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create certificate"})
		return
	}

	encPriv, err := crypto.EncryptPrivateKey(privPEM, h.encKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt private key"})
		return
	}

	parsedCert, err := crypto.ParseCertificate(certPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse certificate"})
		return
	}

	// The certificates.subject / certificates.issuer columns are jsonb
	// (migrations/001_init.sql). A raw DN string ("CN=...,O=...") is not
	// valid JSON, so it MUST be JSON-encoded before being stored — a raw
	// []byte(req.Subject) causes Postgres to reject the INSERT with
	// "invalid input syntax for type json" (SQLSTATE 22P02), discovered
	// via the real-persistence integration test (queue#4, §11.4.27).
	subjectJSON, err := json.Marshal(req.Subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode subject"})
		return
	}
	issuerJSON, err := json.Marshal(ca.CACertPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode issuer"})
		return
	}

	now := time.Now().UTC()
	cert := &model.Certificate{
		ID:           uuid.New(),
		CAID:         caID,
		OrgID:        ca.OrgID,
		Name:         req.Name,
		CertPEM:      certPEM,
		KeyPEM:       encPriv,
		SerialNumber: serial,
		Subject:      subjectJSON,
		Issuer:       issuerJSON,
		NotBefore:    parsedCert.NotBefore,
		NotAfter:     parsedCert.NotAfter,
		Status:       model.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.CreateCert(c.Request.Context(), cert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store certificate"})
		return
	}

	_ = pubPEM
	c.JSON(http.StatusCreated, model.CertResponse{
		ID:           cert.ID,
		CAID:         cert.CAID,
		OrgID:        cert.OrgID,
		Name:         cert.Name,
		CertPEM:      cert.CertPEM,
		SerialNumber: cert.SerialNumber,
		Subject:      cert.Subject,
		Issuer:       cert.Issuer,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Status:       cert.Status,
		CreatedAt:    cert.CreatedAt,
		UpdatedAt:    cert.UpdatedAt,
	})
}

// ListCerts handles listing certificates.
func (h *Handler) ListCerts(c *gin.Context) {
	var req model.ListCertsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var caID, orgID uuid.UUID
	if req.CAID != "" {
		caID = uuid.MustParse(req.CAID)
	}
	if req.OrgID != "" {
		orgID = uuid.MustParse(req.OrgID)
	}

	certs, total, err := h.repo.ListCerts(c.Request.Context(), caID, orgID, req.Status, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list certificates"})
		return
	}

	var resp []*model.CertResponse
	for _, cert := range certs {
		resp = append(resp, &model.CertResponse{
			ID:               cert.ID,
			CAID:             cert.CAID,
			OrgID:            cert.OrgID,
			Name:             cert.Name,
			CertPEM:          cert.CertPEM,
			SerialNumber:     cert.SerialNumber,
			Subject:          cert.Subject,
			Issuer:           cert.Issuer,
			NotBefore:        cert.NotBefore,
			NotAfter:         cert.NotAfter,
			RevokedAt:        cert.RevokedAt,
			RevocationReason: cert.RevocationReason,
			Status:           cert.Status,
			CreatedAt:        cert.CreatedAt,
			UpdatedAt:        cert.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, model.ListCertsResponse{
		Certificates: resp,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
}

// GetCert handles retrieving a single certificate.
func (h *Handler) GetCert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	cert, err := h.repo.GetCertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}

	c.JSON(http.StatusOK, model.CertResponse{
		ID:               cert.ID,
		CAID:             cert.CAID,
		OrgID:            cert.OrgID,
		Name:             cert.Name,
		CertPEM:          cert.CertPEM,
		SerialNumber:     cert.SerialNumber,
		Subject:          cert.Subject,
		Issuer:           cert.Issuer,
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
		RevokedAt:        cert.RevokedAt,
		RevocationReason: cert.RevocationReason,
		Status:           cert.Status,
		CreatedAt:        cert.CreatedAt,
		UpdatedAt:        cert.UpdatedAt,
	})
}

// RevokeCert handles certificate revocation.
func (h *Handler) RevokeCert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.RevokeCert(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke certificate"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "pki-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status (503 if no DB).
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "pki-service",
			"error":   "database not connected",
		})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "pki-service",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "pki-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

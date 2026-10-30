// Package storage fornece um cliente S3-compativel (MinIO local / Cloudflare R2)
// para gerar URLs pre-assinadas de upload e para montar/remover objetos de midia.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Options configura a conexao com o provedor S3-compativel.
type Options struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	PublicBaseURL   string
	UseSSL          bool
	PathStyle       bool
	PresignExpiry   time.Duration
}

// Client encapsula o cliente MinIO/S3 e os metadados do bucket publico.
type Client struct {
	mc            *minio.Client
	bucket        string
	publicBaseURL string
	presignExpiry time.Duration
}

// New cria o cliente. O mesmo codigo atende MinIO (path-style, sem TLS) e
// Cloudflare R2 (virtual-host, TLS), bastando ajustar as Options.
func New(opts Options) (*Client, error) {
	lookup := minio.BucketLookupAuto
	if opts.PathStyle {
		lookup = minio.BucketLookupPath
	}

	mc, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(opts.AccessKeyID, opts.SecretAccessKey, ""),
		Secure:       opts.UseSSL,
		Region:       opts.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar cliente de storage: %w", err)
	}

	expiry := opts.PresignExpiry
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	return &Client{
		mc:            mc,
		bucket:        opts.Bucket,
		publicBaseURL: strings.TrimRight(opts.PublicBaseURL, "/"),
		presignExpiry: expiry,
	}, nil
}

// EnsureBucket garante que o bucket exista e tenha leitura publica (best-effort).
// Em provedores gerenciados (R2) o acesso publico costuma ser configurado fora
// da API, portanto eventuais erros devem ser tratados como aviso.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("nao foi possivel verificar o bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("nao foi possivel criar o bucket: %w", err)
		}
	}

	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Allow",
				"Principal": map[string]interface{}{"AWS": []string{"*"}},
				"Action":    []string{"s3:GetObject"},
				"Resource":  []string{fmt.Sprintf("arn:aws:s3:::%s/*", c.bucket)},
			},
		},
	}
	policyJSON, _ := json.Marshal(policy)
	if err := c.mc.SetBucketPolicy(ctx, c.bucket, string(policyJSON)); err != nil {
		return fmt.Errorf("nao foi possivel definir a politica do bucket: %w", err)
	}
	return nil
}

// EnsureCORS configura as regras de CORS do bucket para permitir o upload via
// PUT pre-assinado direto do navegador (frontend -> storage). E necessario em
// provedores gerenciados (Neon Object Storage), ja que o MinIO local define o
// CORS pela variavel MINIO_API_CORS_ALLOW_ORIGIN. Best-effort.
func (c *Client) EnsureCORS(ctx context.Context, origins []string, methods []string) error {
	if len(origins) == 0 {
		return nil
	}
	if len(methods) == 0 {
		methods = []string{"GET", "PUT", "HEAD"}
	}
	cfg := cors.NewConfig([]cors.Rule{
		{
			AllowedOrigin: origins,
			AllowedMethod: methods,
			AllowedHeader: []string{"*"},
			ExposeHeader:  []string{"ETag"},
			MaxAgeSeconds: 3000,
		},
	})
	return c.mc.SetBucketCors(ctx, c.bucket, cfg)
}

// PresignPut gera uma URL pre-assinada para upload (PUT) de um objeto.
func (c *Client) PresignPut(ctx context.Context, key string) (string, error) {
	u, err := c.mc.PresignedPutObject(ctx, c.bucket, key, c.presignExpiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PublicURL monta a URL publica de leitura de um objeto.
func (c *Client) PublicURL(key string) string {
	return c.publicBaseURL + "/" + strings.TrimLeft(key, "/")
}

// KeyFromURL extrai a chave do objeto a partir da URL publica; retorna "" se a
// URL nao pertencer a este bucket.
func (c *Client) KeyFromURL(publicURL string) string {
	prefix := c.publicBaseURL + "/"
	if !strings.HasPrefix(publicURL, prefix) {
		return ""
	}
	return strings.TrimPrefix(publicURL, prefix)
}

// DeleteObject remove um objeto do bucket (best-effort).
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

// ExpiresInSeconds retorna a validade (em segundos) das URLs pre-assinadas.
func (c *Client) ExpiresInSeconds() int64 {
	return int64(c.presignExpiry.Seconds())
}

package config

import "testing"

func TestApplyNeonStorageDetectsNeon(t *testing.T) {
	t.Setenv("STORAGE_ENDPOINT", "")
	t.Setenv("STORAGE_BUCKET", "")
	t.Setenv("STORAGE_PUBLIC_BASE_URL", "")
	t.Setenv("AWS_ENDPOINT_URL_S3", "https://br-dry-river-b8kfpm7w.storage.c-14.us-east-1.aws.neon.tech")
	t.Setenv("AWS_ACCESS_KEY_ID", "nak_live_test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "nsk_live_test")
	t.Setenv("AWS_REGION", "us-east-1")

	cfg := &Config{
		StorageEndpoint:  "localhost:9000",
		StorageBucket:    "fitconecta-media",
		StorageProvision: true,
	}
	applyNeonStorage(cfg)

	if cfg.StorageEndpoint != "br-dry-river-b8kfpm7w.storage.c-14.us-east-1.aws.neon.tech" {
		t.Errorf("endpoint = %q, quero o host do Neon sem esquema", cfg.StorageEndpoint)
	}
	if !cfg.StorageUseSSL {
		t.Error("UseSSL = false, o endpoint https do Neon exige TLS")
	}
	if !cfg.StoragePathStyle {
		t.Error("PathStyle = false, o Neon aceita apenas path-style")
	}
	if cfg.StorageRegion != "us-east-1" {
		t.Errorf("region = %q, quero AWS_REGION", cfg.StorageRegion)
	}
	if cfg.StorageAccessKeyID != "nak_live_test" || cfg.StorageSecretAccessKey != "nsk_live_test" {
		t.Error("credenciais AWS do Neon nao foram aplicadas")
	}
	wantPublic := "https://br-dry-river-b8kfpm7w.storage.c-14.us-east-1.aws.neon.tech/fitconecta-media"
	if cfg.StoragePublicBaseURL != wantPublic {
		t.Errorf("publicBaseURL = %q, quero %q", cfg.StoragePublicBaseURL, wantPublic)
	}
	if cfg.StorageProvision {
		t.Error("StorageProvision = true, o bucket do Neon e gerenciado fora da API")
	}
}

func TestApplyNeonStorageHonorsExplicitEndpoint(t *testing.T) {
	t.Setenv("STORAGE_ENDPOINT", "minio.localhost:9000")
	t.Setenv("AWS_ENDPOINT_URL_S3", "https://neon.storage.example")

	cfg := &Config{
		StorageEndpoint:    "minio.localhost:9000",
		StorageAccessKeyID: "fitconecta_minio",
		StorageProvision:   true,
	}
	applyNeonStorage(cfg)

	if cfg.StorageEndpoint != "minio.localhost:9000" {
		t.Errorf("endpoint = %q, STORAGE_ENDPOINT explicito deve ter prioridade", cfg.StorageEndpoint)
	}
	if cfg.StorageAccessKeyID != "fitconecta_minio" {
		t.Error("credenciais MinIO foram sobrescritas")
	}
	if !cfg.StorageProvision {
		t.Error("StorageProvision = false, em MinIO a API deve provisionar o bucket")
	}
}

func TestApplyNeonStorageWithoutNeonVars(t *testing.T) {
	t.Setenv("STORAGE_ENDPOINT", "")
	t.Setenv("AWS_ENDPOINT_URL_S3", "")

	cfg := &Config{
		StorageEndpoint:  "localhost:9000",
		StorageProvision: true,
	}
	applyNeonStorage(cfg)

	if cfg.StorageEndpoint != "localhost:9000" {
		t.Errorf("endpoint = %q, sem variaveis do Neon o padrao deve permanecer", cfg.StorageEndpoint)
	}
	if !cfg.StorageProvision {
		t.Error("StorageProvision = false, sem Neon a API deve provisionar o bucket")
	}
}

func TestApplyNeonStorageProviderNeonOverridesEndpoint(t *testing.T) {
	t.Setenv("STORAGE_PROVIDER", "neon")
	t.Setenv("STORAGE_ENDPOINT", "minio.localhost:9000")
	t.Setenv("STORAGE_BUCKET", "")
	t.Setenv("STORAGE_PUBLIC_BASE_URL", "")
	t.Setenv("AWS_ENDPOINT_URL_S3", "https://neon.storage.example")
	t.Setenv("AWS_ACCESS_KEY_ID", "nak_live_test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "nsk_live_test")

	cfg := &Config{StorageEndpoint: "minio.localhost:9000", StorageProvision: true}
	applyNeonStorage(cfg)

	if cfg.StorageEndpoint != "neon.storage.example" {
		t.Errorf("endpoint = %q, com STORAGE_PROVIDER=neon o endpoint do Neon deve prevalecer", cfg.StorageEndpoint)
	}
	if cfg.StorageProvision {
		t.Error("StorageProvision = false era esperado com o Neon")
	}
}

func TestApplyNeonStorageProviderMinioIgnoresNeon(t *testing.T) {
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("AWS_ENDPOINT_URL_S3", "https://neon.storage.example")

	cfg := &Config{
		StorageEndpoint:    "minio.localhost:9000",
		StorageAccessKeyID: "fitconecta_minio",
		StorageProvision:   true,
	}
	applyNeonStorage(cfg)

	if cfg.StorageEndpoint != "minio.localhost:9000" {
		t.Errorf("endpoint = %q, com STORAGE_PROVIDER=minio as variaveis STORAGE_* valem", cfg.StorageEndpoint)
	}
	if cfg.StorageAccessKeyID != "fitconecta_minio" {
		t.Error("credenciais MinIO foram sobrescritas pelo Neon")
	}
	if !cfg.StorageProvision {
		t.Error("StorageProvision = false em MinIO")
	}
}

func TestSplitS3Endpoint(t *testing.T) {
	cases := []struct {
		in     string
		host   string
		useSSL bool
	}{
		{"https://s3.example.com", "s3.example.com", true},
		{"http://minio.localhost:9000", "minio.localhost:9000", false},
		{"https://s3.example.com/", "s3.example.com", true},
		{"s3.example.com", "s3.example.com", true},
	}

	for _, c := range cases {
		host, useSSL := splitS3Endpoint(c.in)
		if host != c.host || useSSL != c.useSSL {
			t.Errorf("splitS3Endpoint(%q) = (%q, %v), quero (%q, %v)", c.in, host, useSSL, c.host, c.useSSL)
		}
	}
}

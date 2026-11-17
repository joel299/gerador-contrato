package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables and optional .env file.
type Config struct {
	Port        string
	Env         string
	DBPath      string
	LocalPDFDir string

	S3Endpoint     string
	S3Bucket       string
	S3Region       string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool

	PDFPoolSize int
	ChromePath  string

	ContratadaRazaoSocial   string
	ContratadaCNPJ          string
	ContratadaEndereco      string
	ContratadaRepresentante string
	ContratadaCPF           string
	DefaultCityForum        string
}

// Load reads .env (if present) and populates Config from environment variables with safe defaults.
func Load() *Config {
	loadDotEnv(".env")

	poolSize := getEnvInt("PDF_POOL_SIZE", 2)
	if poolSize < 1 {
		poolSize = 1
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DBPath:      getEnv("DB_PATH", "./data/contracts.db"),
		LocalPDFDir: getEnv("LOCAL_PDF_DIR", "./data/pdfs"),

		S3Endpoint:     os.Getenv("S3_ENDPOINT"),
		S3Bucket:       os.Getenv("S3_BUCKET"),
		S3Region:       getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:    os.Getenv("S3_SECRET_KEY"),
		S3UsePathStyle: getEnvBool("S3_USE_PATH_STYLE", true),

		PDFPoolSize: poolSize,
		ChromePath:  os.Getenv("CHROME_PATH"),

		ContratadaRazaoSocial:   getEnv("CONTRATADA_RAZAO_SOCIAL", "JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP"),
		ContratadaCNPJ:          getEnv("CONTRATADA_CNPJ", "63741075000143"),
		ContratadaEndereco:      getEnv("CONTRATADA_ENDERECO", "Campo Grande/MS"),
		ContratadaRepresentante: getEnv("CONTRATADA_REPRESENTANTE", "Joel Acosta Quintana Junior"),
		ContratadaCPF:           os.Getenv("CONTRATADA_CPF"),
		DefaultCityForum:        getEnv("DEFAULT_CITY_FORUM", "Campo Grande/MS"),
	}
}

// HasS3 returns true when minimum S3 credentials and bucket are configured.
func (c *Config) HasS3() bool {
	return strings.TrimSpace(c.S3Bucket) != "" &&
		strings.TrimSpace(c.S3AccessKey) != "" &&
		strings.TrimSpace(c.S3SecretKey) != ""
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

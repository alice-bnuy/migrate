package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// LoadRefreshTokenFromEnv carrega o refresh token do arquivo .env
func LoadRefreshTokenFromEnv() (string, error) {
	// Tenta carregar o .env
	if err := godotenv.Load(); err != nil {
		return "", fmt.Errorf("erro ao carregar .env: %w", err)
	}

	refreshToken := os.Getenv("GOOGLE_REFRESH_TOKEN")
	if refreshToken == "" {
		return "", fmt.Errorf("GOOGLE_REFRESH_TOKEN não encontrado no .env")
	}

	return refreshToken, nil
}

// GenerateOAuthToken cria um token OAuth completo a partir do refresh token
func GenerateOAuthToken(credentialsFile string, refreshToken string, scopes []string) (*oauth2.Token, error) {
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler credenciais: %w", err)
	}

	config, err := google.ConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear credenciais: %w", err)
	}

	// Cria um token inicial com o refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour), // Força refresh imediato
	}

	// Usa o TokenSource para obter um access token válido
	tokenSource := config.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter access token: %w", err)
	}

	return newToken, nil
}

// SaveTokenToFile salva o token OAuth em formato JSON
func SaveTokenToFile(token *oauth2.Token, filePath string) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("falha ao serializar token: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("falha ao salvar arquivo: %w", err)
	}

	return nil
}

// RunOAuthTokenFlow executa o fluxo completo para gerar token OAuth
func RunOAuthTokenFlow() error {
	const credentialsFile = "client_secret_2_601804493169-nh1uc56rqsuco7f2f7saplpjg21tijse.apps.googleusercontent.com.json"
	const tokenFile = "token.json"
	const scopes = "https://www.googleapis.com/auth/drive"

	fmt.Println("🔑 GERAR TOKEN OAUTH DO GOOGLE DRIVE")
	fmt.Println(strings.Repeat("=", 50))

	// Carrega o refresh token do .env
	fmt.Println("📂 Carregando refresh token do .env...")
	refreshToken, err := LoadRefreshTokenFromEnv()
	if err != nil {
		return fmt.Errorf("erro ao carregar refresh token: %w", err)
	}
	fmt.Println("✅ Refresh token carregado com sucesso")

	// Gera o token OAuth completo
	fmt.Println("🔄 Gerando token OAuth...")
	token, err := GenerateOAuthToken(credentialsFile, refreshToken, []string{scopes})
	if err != nil {
		return fmt.Errorf("erro ao gerar token OAuth: %w", err)
	}
	fmt.Println("✅ Token OAuth gerado com sucesso")

	// Salva o token no arquivo
	fmt.Printf("💾 Salvando token em %s...\n", tokenFile)
	if err := SaveTokenToFile(token, tokenFile); err != nil {
		return fmt.Errorf("erro ao salvar token: %w", err)
	}
	fmt.Printf("✅ Token salvo em %s\n", tokenFile)

	// Exibe informações do token
	fmt.Println("\n📋 INFORMAÇÕES DO TOKEN:")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Access Token: %s...%s\n", token.AccessToken[:20], token.AccessToken[len(token.AccessToken)-10:])
	fmt.Printf("Token Type: %s\n", token.TokenType)
	fmt.Printf("Expira em: %s\n", token.Expiry.Format("2006-01-02 15:04:05"))
	fmt.Printf("Válido por: %s\n", time.Until(token.Expiry).Round(time.Minute))

	if token.RefreshToken != "" {
		fmt.Printf("Refresh Token: %s...%s\n", token.RefreshToken[:20], token.RefreshToken[len(token.RefreshToken)-10:])
	}

	fmt.Println("\n💡 PRÓXIMOS PASSOS:")
	fmt.Printf("1. Use o arquivo %s em suas aplicações\n", tokenFile)
	fmt.Println("2. O token será renovado automaticamente quando expirar")
	fmt.Println("3. Mantenha o arquivo seguro (não faça commit no Git)")

	return nil
}

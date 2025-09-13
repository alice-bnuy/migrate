package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// randomState gera um estado aleatório para proteção CSRF
func randomState(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GetRefreshToken executa o fluxo OAuth 2.0 para obter um refresh token
func GetRefreshToken(credentialsFile string, scopes []string) (string, error) {
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return "", fmt.Errorf("falha ao ler credenciais: %w", err)
	}

	config, err := google.ConfigFromJSON(data, scopes...)
	if err != nil {
		return "", fmt.Errorf("falha ao parsear credenciais: %w", err)
	}

	state := randomState(12)

	// prompt=consent força re-exibir consentimento e aumenta chance de vir refresh token
	authURL := config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	fmt.Println("🔐 OBTER REFRESH TOKEN DO GOOGLE DRIVE")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("1) Abra a URL abaixo no navegador e autorize o acesso:")
	fmt.Println("   " + authURL)
	fmt.Println()
	fmt.Println("2) Depois da autorização aparecerá um erro em localhost (isso é esperado).")
	fmt.Println("3) Copie o valor do parâmetro 'code' da URL (não inclua '&scope=...').")
	fmt.Print("\n📝 Cole aqui o código de autorização: ")

	var raw string
	if _, err := fmt.Scan(&raw); err != nil {
		return "", fmt.Errorf("falha ao ler código: %w", err)
	}

	// Às vezes a pessoa cola a URL inteira. Extrair se necessário.
	authCode := raw
	if strings.Contains(raw, "code=") {
		parts := strings.Split(raw, "code=")
		if len(parts) > 1 {
			authCode = parts[1]
			if i := strings.Index(authCode, "&"); i >= 0 {
				authCode = authCode[:i]
			}
		}
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return "", fmt.Errorf("falha ao trocar código por token: %w", err)
	}

	if tok.RefreshToken == "" {
		return "", fmt.Errorf("nenhum refresh token retornado. Revogue o acesso em https://myaccount.google.com/permissions e tente de novo (ou verifique se usou prompt=consent)")
	}

	return tok.RefreshToken, nil
}

// RunRefreshTokenFlow executa o fluxo completo para obter refresh token
func RunRefreshTokenFlow() error {
	const credentialsFile = "client_secret_2_601804493169-nh1uc56rqsuco7f2f7saplpjg21tijse.apps.googleusercontent.com.json"
	const scopes = "https://www.googleapis.com/auth/drive"

	refreshToken, err := GetRefreshToken(credentialsFile, []string{scopes})
	if err != nil {
		return fmt.Errorf("erro ao obter refresh token: %w", err)
	}

	fmt.Println("\n✅ REFRESH TOKEN OBTIDO COM SUCESSO!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("📋 Refresh Token:")
	fmt.Println(refreshToken)
	fmt.Println()
	fmt.Println("💡 PRÓXIMOS PASSOS:")
	fmt.Println("1. Copie o refresh token acima")
	fmt.Println("2. Adicione no arquivo .env como:")
	fmt.Println("   GOOGLE_REFRESH_TOKEN=" + refreshToken)
	fmt.Println("3. Execute 'setup oauth_token' para gerar o token OAuth completo")

	return nil
}

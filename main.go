package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Timeout para as requisições
	timeout = 1 * time.Second
)

// Estrutura para armazenar a resposta de cada API
type ApiResponse struct {
	API        string `json:"api"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Cidade     string `json:"localidade"`
	Uf         string `json:"uf"`
}

func main() {
	var cep string
	fmt.Print("Digite o CEP: ")
	fmt.Scanln(&cep)

	// Verifica se o CEP tem o formato correto (ex: 01153000)
	if len(cep) != 8 || !isNumeric(cep) {
		fmt.Println("CEP inválido!")
		return
	}

	// Prepara as URLs das APIs
	brasilAPIUrl := fmt.Sprintf("https://brasilapi.com.br/api/cep/v1/%s", cep)
	viaCepUrl := fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep)

	// Criar um contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Canal para pegar o resultado da API mais rápida
	resultChannel := make(chan ApiResponse, 1)

	// Fazendo as requisições simultâneas
	go func() {
		getCepData(ctx, brasilAPIUrl, "BrasilAPI", resultChannel)
	}()

	go func() {
		getCepData(ctx, viaCepUrl, "ViaCEP", resultChannel)
	}()

	// Esperar a primeira resposta ou timeout
	select {
	case result := <-resultChannel:
		// Exibe o resultado da API que respondeu mais rápido
		fmt.Printf("Resposta mais rápida foi da %s:\n", result.API)
		fmt.Printf("Logradouro: %s\nBairro: %s\nCidade: %s\nUF: %s\n", result.Logradouro, result.Bairro, result.Cidade, result.Uf)
	case <-ctx.Done():
		// Se o contexto for cancelado, isso significa que houve timeout
		fmt.Println("Erro: Tempo de resposta excedido. Timeout de 1 segundo.")
	}
}

// Função para fazer a requisição e parse da resposta
func getCepData(ctx context.Context, url, apiName string, resultChannel chan<- ApiResponse) {
	// Criar o cliente HTTP com o mesmo contexto de timeout
	client := &http.Client{}

	// Faz a requisição HTTP com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}

	// Realiza a requisição
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	// Verifica se a resposta é válida
	if resp.StatusCode != http.StatusOK {
		fmt.Println(err)
		return
	}

	// Lê o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Parse o JSON de resposta
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(data) < 4 {
		fmt.Printf("Resposta inesperada do sevidor %s \n", apiName)
		time.Sleep(time.Millisecond * 2)
		return
	}

	// Converte os dados para o formato desejado
	result := getApiResponse(apiName, data)

	// Envia o resultado ao canal
	select {
	case resultChannel <- result:
		// Enviou a resposta para o canal
		return
	case <-ctx.Done():
		// Caso o contexto tenha sido cancelado (tempo de espera excedido)
		return
	}
}

// Função para converter um mapa de sting na struct ApiResponse
func getApiResponse(apiName string, data map[string]interface{}) ApiResponse {
	var result ApiResponse
	if apiName == "BrasilAPI" {
		result = ApiResponse{
			API:        "BrasilAPI",
			Logradouro: data["street"].(string),
			Bairro:     data["neighborhood"].(string),
			Cidade:     data["city"].(string),
			Uf:         data["state"].(string),
		}
	} else if apiName == "ViaCEP" {
		result = ApiResponse{
			API:        "ViaCEP",
			Logradouro: data["logradouro"].(string),
			Bairro:     data["bairro"].(string),
			Cidade:     data["localidade"].(string),
			Uf:         data["uf"].(string),
		}
	}
	return result
}

// Função para verificar se o CEP contém apenas números
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

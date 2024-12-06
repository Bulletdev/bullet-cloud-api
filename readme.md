# API RESTful em Go para gerenciamento de produtos

<p align="center">
  <img alt="GitHub top language" src="https://img.shields.io/github/languages/top/Bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">
  
  <a href="https://www.linkedin.com/in/Michael-Bullet/">
    <img alt="Made by" src="https://img.shields.io/static/v1?label=made%20by&message=Michael%20Bullet&color=04D361&labelColor=000000">
  </a>
  
  <img alt="Repository size" src="https://img.shields.io/github/repo-size/bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">
  
  <a href="https://github.com/Bulletdev/linktree/commits/master">
    <img alt="GitHub last commit" src="https://img.shields.io/github/last-commit/bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">
  </a>
</p>

[![CodeQL Advanced](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml/badge.svg)](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml)

✨ Recursos
<div>
CRUD completo de produtos
</div> 
  
<div> 
Armazenamento in-memory
</div> 

<div>
Tratamento seguro de concorrência
</div> 

<div>
Endpoints RESTful
</div> 

Health check
</div> 

<div> 
Testes Unitários 

</div> 

## demonstração: 

<details>
<img src="/demo.png">
</details>

🛠 Tecnologias

<div>
Golang
</div> 

<div>  
Gorilla Mux
</div> 

<div>
Sync primitives
</div> 



## 📦 Instalação

Pré-requisitos

Go 1.21+
Git

Passos
```bash
git clone https://github.com/bulletdev/bullet-cloud-api.git
```
# Entrar no diretório
cd bullet-cloud-api

# Baixar dependências
go mod tidy

# Rodar aplicação
go run cmd/main.go





🔍 Endpoints

<div>
  
GET /products: Listar todos produtos

POST /products: Criar produto

GET /products/{id}: Buscar produto específico

PUT /products/{id}: Atualizar produto

DELETE /products/{id}: Deletar produto

GET /health: Verificar status da aplicação

</div> 

🧪 Test

<details>
<img src="/teste-ok.png">
</details>

```bash

go test ./...
```

📄 Licença

BulletDEv all rights reserveds




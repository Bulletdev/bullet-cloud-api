<p align="center">
  
[![CodeQL Advanced](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml/badge.svg)](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml)
[![Go](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/go.yml/badge.svg)](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/go.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Bulletdev_Arremate-certo&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Bulletdev_Arremate-certo)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=Bulletdev_Arremate-certo&metric=bugs)](https://sonarcloud.io/summary/new_code?id=Bulletdev_Arremate-certo)
<img src="https://img.shields.io/badge/status-Produção-Blue"> 
</p>     
   
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

# ✨ Recursos
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

<div>
Health check
</div> 

<div> 
Testes Unitários 

</div> 


## 🚀 Exemplo de uso

 Windows: 
 
```bash
Invoke-WebRequest -Uri http://localhost:4444/products -Method POST -Headers @{"Content-Type"="application/json"} -Body '{"name":"Notebook Gamer","description":"Notebook para jogos","price":5999.99,"category":"Eletronicos"}'
```
No linux: 

```bash
curl -X POST http://localhost:4444/products \
-H "Content-Type: application/json" \
-d '{"name":"Notebook Gamer","description":"Notebook para jogos","price":5999.99,"category":"Eletronicos"}'
```
## Documentação da API

Para acessar a documentação completa da API, clique no link abaixo:

[Documentação da API no Swagger](https://app.swaggerhub.com/apis-docs/bulletcloud/Estoque/1.1) <! -- CTRL + click para abrir em nova aba -->



## demonstração: 

<details>
<img src="/public/teste2.jpeg">
</details>


## 🛠 Tecnologias

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





## 🔍 Endpoints

<div>
  
GET /products: Listar todos produtos

POST /products: Criar produto

GET /products/{id}: Buscar produto específico

PUT /products/{id}: Atualizar produto

DELETE /products/{id}: Deletar produto

GET /health: Verificar status da aplicação

</div> 

## 🧪 Test

<details>
<img src="/public/demo2.png">
</details>

```bash

go test ./...
```

📄 Licença

BulletDEv all rights reserveds




<p align="center">
  
[![CodeQL Advanced](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml/badge.svg)](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/codeql.yml)
[![Go](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/go.yml/badge.svg)](https://github.com/Bulletdev/bullet-cloud-api/actions/workflows/go.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Bulletdev_Arremate-certo&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Bulletdev_Arremate-certo)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=Bulletdev_Arremate-certo&metric=bugs)](https://sonarcloud.io/summary/new_code?id=Bulletdev_Arremate-certo)
<img src="https://img.shields.io/badge/status-Em%20Desenvolvimento-Orange"> 
</p>     
   
# API RESTful em Go para E-commerce (Bullet Cloud API)
 
<p align="center"> 
  <img alt="GitHub top language" src="https://img.shields.io/github/languages/top/Bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">  
  
  <a href="https://www.linkedin.com/in/Michael-Bullet/">
    <img alt="Made by" src="https://img.shields.io/static/v1?label=made%20by&message=Michael%20Bullet&color=04D361&labelColor=000000">
  </a>  
  
  <img alt="Repository size" src="https://img.shields.io/github/repo-size/bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">
  
  <a href="https://github.com/Bulletdev/bullet-cloud-api/commits/master">
    <img alt="GitHub last commit" src="https://img.shields.io/github/last-commit/bulletdev/bullet-cloud-api?color=04D361&labelColor=000000">
  </a>
</p>

# ✨ Recursos Atuais e Planejados
<div>
Autenticação e Gerenciamento de Usuários (Registro, Login, Dados do Usuário, Endereços)
</div>  
<div>
Gerenciamento de Produtos e Categorias
</div>
<div>
Armazenamento de dados com PostgreSQL (via Supabase)
</div> 
<div>
Autenticação segura com JWT e Hashing de Senha (bcrypt)
</div> 
<div>
Endpoints RESTful com prefixo `/api`
</div> 
<div>
Health check
</div> 
<div> 
Testes Unitários (Existentes/Planejados)
</div> 
<div>
*Planejado:* Carrinho, Pedidos, Frete, Paginação, Filtros, Validação Avançada, Permissões (Admin)
</div>

## 🚀 Exemplo de uso

(Veja a seção Endpoints Atuais para mais detalhes)

**Registrar um novo usuário:**

*Windows (PowerShell):*
```powershell
Invoke-RestMethod -Uri http://localhost:4444/api/auth/register -Method POST -ContentType "application/json" -Body '{"name":"Nome Sobrenome","email":"email@exemplo.com","password":"senha123"}'
```
*Linux/macOS (curl):*
```bash
curl -X POST http://localhost:4444/api/auth/register \
-H "Content-Type: application/json" \
-d '{"name":"Nome Sobrenome","email":"email@exemplo.com","password":"senha123"}'
```

**Fazer login:**

*Windows (PowerShell):*
```powershell
$response = Invoke-RestMethod -Uri http://localhost:4444/api/auth/login -Method POST -ContentType "application/json" -Body '{"email":"email@exemplo.com","password":"senha123"}'
$token = $response.token
Write-Host "Token JWT: $token"
```
*Linux/macOS (curl) (requer `jq` para extrair o token):*
```bash
TOKEN=$(curl -s -X POST http://localhost:4444/api/auth/login \
-H "Content-Type: application/json" \
-d '{"email":"email@exemplo.com","password":"senha123"}' | jq -r .token)
echo "Token JWT: $TOKEN"
```

**Adicionar um endereço (requer token):**

*Linux/macOS (curl) (assumindo que USER_ID e TOKEN estão definidos):*
```bash
curl -X POST http://localhost:4444/api/users/$USER_ID/addresses \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{"street":"Rua Exemplo, 123","city":"Cidade","state":"SP","postal_code":"12345-678","country":"Brasil","is_default":true}'
```

## Documentação da API (Planejada)

(A documentação Swagger existente pode estar desatualizada. Será atualizada conforme a API evolui.)

[Documentação da API no Swagger](https://app.swaggerhub.com/apis-docs/bulletcloud/Estoque/1.1) 


## 🛠 Tecnologias

<div>
Golang
</div> 
<div>  
Gorilla Mux
</div> 
<div>
PostgreSQL (via Supabase)
</div>
<div>
pgx (Driver PostgreSQL)
</div>
<div>
JWT (github.com/golang-jwt/jwt/v5)
</div>
<div>
bcrypt (Hashing de Senha)
</div>
<div>
golang-migrate/migrate (Migrações de Banco de Dados)
</div>


## 📦 Instalação

**Pré-requisitos**

*   Go (versão especificada no `go.mod`, ex: 1.21+)
*   Git
*   Docker (Opcional, para rodar banco localmente se não usar Supabase)
*   [golang-migrate/migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) (Instalada e no PATH)

**Passos**

1.  **Clonar o repositório:**
    ```bash
    git clone https://github.com/bulletdev/bullet-cloud-api.git
    cd bullet-cloud-api
    ```
2.  **Configurar Variáveis de Ambiente:**
    *   Crie um arquivo chamado `.env` na raiz do projeto.
    *   Adicione as seguintes variáveis, substituindo pelos seus valores:
        ```env
        # URL de conexão do seu banco PostgreSQL (ex: Supabase)
        DATABASE_URL=postgres://usuario:senha@host:porta/database?sslmode=require
        
        # Segredo para assinar os tokens JWT (obtenha do Supabase ou gere um seguro)
        JWT_SECRET=seu_segredo_super_seguro_aqui 
        
        # Porta da API (opcional, padrão 4444)
        # API_PORT=4444 
        ```
    *   **Importante:** Adicione `.env` ao seu `.gitignore` (já deve estar feito).
3.  **Instalar Dependências:**
    ```bash
    go mod tidy
    go mod vendor # Se estiver usando vendoring
    ```
4.  **Aplicar Migrações do Banco:**
    *   Certifique-se que a CLI `migrate` está instalada.
    *   Execute (substitua a URL se não estiver usando `.env` diretamente):
        ```bash
        # No Linux/macOS (se .env carregado no shell)
        # migrate -database ${DATABASE_URL} -path internal/database/migrations up
        
        # No Windows PowerShell (se .env carregado no shell)
        # migrate -database $env:DATABASE_URL -path internal/database/migrations up
        
        # Passando a URL diretamente (mais seguro se .env não carregado)
        migrate -database 'SUA_DATABASE_URL_COMPLETA_AQUI' -path internal/database/migrations up 
        ```
5.  **Rodar Aplicação:**
    ```bash
    go run cmd/main.go
    ```

## 🔍 Endpoints Atuais

**Saúde**
*   `GET /api/health`: Verifica status da aplicação.

**Autenticação**
*   `POST /api/auth/register`: Registra um novo usuário.
*   `POST /api/auth/login`: Autentica um usuário e retorna um token JWT.

**Usuários**
*   `GET /api/users/me` (Protegido): Retorna informações do usuário autenticado.
*   `GET /api/users/{userId}/addresses` (Protegido): Lista endereços do usuário especificado.
*   `POST /api/users/{userId}/addresses` (Protegido): Adiciona um novo endereço para o usuário.
*   `PUT /api/users/{userId}/addresses/{addressId}` (Protegido): Atualiza um endereço existente do usuário.
*   `DELETE /api/users/{userId}/addresses/{addressId}` (Protegido): Remove um endereço do usuário.
*   `PATCH /api/users/{userId}/addresses/{addressId}/default` (Protegido): Define um endereço como padrão.

**Produtos**
*   `GET /api/products`: Lista todos os produtos.
*   `GET /api/products/{id}`: Busca um produto específico.
*   `POST /api/products` (Protegido): Cria um novo produto.
*   `PUT /api/products/{id}` (Protegido): Atualiza um produto existente.
*   `DELETE /api/products/{id}` (Protegido): Deleta um produto.

**Categorias**
*   `GET /api/categories`: Lista todas as categorias.
*   `GET /api/categories/{id}`: Busca uma categoria específica.
*   `POST /api/categories` (Protegido): Cria uma nova categoria.
*   `PUT /api/categories/{id}` (Protegido): Atualiza uma categoria existente.
*   `DELETE /api/categories/{id}` (Protegido): Deleta uma categoria.

*(Carrinho, Pedidos, Frete serão adicionados futuramente)*


## 🧪 Test

(Instruções de teste podem precisar de atualização)

```bash
go test ./...
```

📄 Licença

BulletDEv all rights reserveds

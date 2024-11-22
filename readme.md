# API RESTful em Go para gerenciamento de produtos

✨ Recursos

CRUD completo de produtos
Armazenamento in-memory
Tratamento seguro de concorrência
Endpoints RESTful
Health check

🛠 Tecnologias

Golang
Gorilla Mux
Sync primitives

📦 Instalação
Pré-requisitos

Go 1.21+
Git

Passos
bashCopy# Clonar repositório
git clone https://github.com/bulletdev/magalu-cloud-api.git

# Entrar no diretório
cd magalu-cloud-api

# Baixar dependências
go mod tidy

# Rodar aplicação
go run cmd/main.go


🔍 Endpoints

GET /products: Listar todos produtos
POST /products: Criar produto
GET /products/{id}: Buscar produto específico
PUT /products/{id}: Atualizar produto
DELETE /products/{id}: Deletar produto
GET /health: Verificar status da aplicação

🧪 Testes

bashCopygo test ./...


📄 Licença

BulletDEv all rights reserveds

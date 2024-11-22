# API RESTful em Go para gerenciamento de produtos

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
git clone https://github.com/bulletdev/magalu-cloud-api.git
```
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

🧪 Test

```bash

go test ./...
```

📄 Licença

BulletDEv all rights reserveds


<img src="/git-api.png">

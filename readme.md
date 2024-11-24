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
git clone https://github.com/bulletdev/magalu-cloud-api.git
```
# Entrar no diretório
cd magalu-cloud-api

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
<img src="/test-ok.png">
</details>

```bash

go test ./...
```

📄 Licença

BulletDEv all rights reserveds




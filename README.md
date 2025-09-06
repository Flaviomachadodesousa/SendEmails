# 📧 Envio de E-mails em Massa com Go

Este projeto é uma aplicação em **Go** para envio de e-mails em massa, utilizando **PostgreSQL**, **GORM**, **gomail** e templates HTML.  
Ele foi projetado para buscar usuários no banco de dados, renderizar mensagens personalizadas e enviá-las de forma concorrente através de múltiplos workers.

---

## 🚀 Tecnologias Utilizadas
- [Go](https://go.dev/)  
- [GORM](https://gorm.io/) – ORM para PostgreSQL  
- [GoMail](https://pkg.go.dev/gopkg.in/gomail.v2) – Envio de e-mails via SMTP  
- [godotenv](https://github.com/joho/godotenv) – Carregar variáveis de ambiente  
- PostgreSQL  
- Templates HTML para personalização dos e-mails  

---

## 📂 Estrutura do Projeto
├── main.go # Arquivo principal
├── models/ # Modelos do banco (Usuario, EmailStatus, etc.)
├── templates/
│ └── email.html # Template HTML do e-mail
├── .env # Variáveis de ambiente (não versionar)
└── README.md # Documenta

---

## ⚙️ Configuração

Crie um arquivo **`.env`** na raiz do projeto com as seguintes variáveis:

# Banco de dados
DB_HOST=localhost\
DB_PORT=5432\
DB_USER=seu_usuario\
DB_PASSWORD=sua_senha\
DB_NAME=seu_banco

# SMTP
SMTP_HOST=smtp.seuprovedor.com
SMTP_PORT=587
SMTP_EMAIL=seuemail@provedor.com
SMTP_PASSWORD=suasenha

# Workers
NUM_WORKERS=5

## ⚙️ Como rodar o projeto
git clone
go mod tidy # Instale as dependências
go run main.go # Execute o projeto

## 📬 Funcionamento

1.O sistema conecta ao banco de dados PostgreSQL.
2.Carrega os usuários da tabela usuarios.
3.Para cada usuário, renderiza o template HTML (templates/email.html).
4.Envia os e-mails em paralelo usando múltiplos workers.
5.Registra o status de cada envio na tabela email_status.
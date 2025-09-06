package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"sync"

	"sendemail/models"

	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config armazena todas as configurações da aplicação.
type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	SMTPHost     string
	SMTPPort     int
	SMTPEmail    string
	SMTPPassword string
	NumWorkers   int
}

// loadConfig carrega as configurações das variáveis de ambiente a partir de um arquivo .env.
func loadConfig() Config {
	godotenv.Load() // Carrega o arquivo .env

	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))     // Carrega o variavies do arquivo .env
	numWorkers, _ := strconv.Atoi(os.Getenv("NUM_WORKERS")) // Carrega o variavies do arquivo .env

	return Config{
		DBHost:       os.Getenv("DB_HOST"),
		DBPort:       os.Getenv("DB_PORT"),
		DBUser:       os.Getenv("DB_USER"),
		DBPassword:   os.Getenv("DB_PASSWORD"),
		DBName:       os.Getenv("DB_NAME"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPEmail:    os.Getenv("SMTP_EMAIL"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		NumWorkers:   numWorkers,
	}
}

// EmailSender encapsula as dependências para o envio de e-mails.
type EmailSender struct {
	db     *gorm.DB
	tmpl   *template.Template
	dialer *gomail.Dialer
	config Config
}

// NewEmailSender cria uma nova instância do EmailSender.
func NewEmailSender(db *gorm.DB, tmpl *template.Template, config Config) *EmailSender {
	dialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPEmail, config.SMTPPassword)
	return &EmailSender{db: db, tmpl: tmpl, dialer: dialer, config: config}
}

func main() {
	fmt.Println("==== Envio de E-mails em Massa ====")
	config := loadConfig()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Erro ao conectar no banco:", err)
	}
	fmt.Println("✅ Conexão ao banco realizada.")

	// ▶️ USA A SUA FUNÇÃO `AutoMigrate` DO PACOTE `models`
	models.AutoMigrate(db)

	tmpl, err := template.ParseFiles("templates/email.html")
	if err != nil {
		log.Fatal("❌ Erro ao carregar template:", err)
	}

	sender := NewEmailSender(db, tmpl, config)
	sender.DispatchAndSend()

	fmt.Println("\nProcesso finalizado. Pressione ENTER para sair...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// DispatchAndSend busca usuários e dispara os workers.
func (s *EmailSender) DispatchAndSend() {
	// ▶️ USA O SEU STRUCT `models.Usuario`
	var usuarios []models.Usuario
	if err := s.db.Order("id").Find(&usuarios).Error; err != nil {
		log.Println("Erro ao buscar usuários:", err)
		return
	}

	if len(usuarios) == 0 {
		log.Println("Nenhum usuário encontrado para envio.")
		return
	}

	fmt.Printf("Encontrados %d usuários. Iniciando envio...\n", len(usuarios))

	// ▶️ O CANAL TRANSPORTA OBJETOS `models.Usuario`
	jobs := make(chan models.Usuario, len(usuarios))
	var wg sync.WaitGroup

	for w := 0; w < s.config.NumWorkers; w++ {
		wg.Add(1)
		go s.worker(w+1, jobs, &wg)
	}

	for _, u := range usuarios {
		jobs <- u
	}
	close(jobs)

	wg.Wait()
}

// worker processa e envia os e-mails do canal de jobs.
func (s *EmailSender) worker(id int, jobs <-chan models.Usuario, wg *sync.WaitGroup) {
	defer wg.Done()
	// ▶️ `u` É UMA INSTÂNCIA DO SEU STRUCT `models.Usuario`
	for u := range jobs {
		// ▶️ `u.Email` É USADO PARA LOGS E PARA O REGISTRO DE STATUS
		if err := s.sendSingleEmail(u); err != nil {
			log.Printf("[Worker %d] ❌ Erro ao enviar para %s: %v", id, u.Email, err)
			s.registerStatus(u.Email, "erro")
		} else {
			log.Printf("[Worker %d] ✅ E-mail enviado para %s", id, u.Email)
			s.registerStatus(u.Email, "sucesso")
		}
	}
}

// sendSingleEmail compõe e envia um único e-mail.
func (s *EmailSender) sendSingleEmail(u models.Usuario) error {
	var body bytes.Buffer
	// ▶️ PASSA O STRUCT `u` INTEIRO PARA O TEMPLATE
	if err := s.tmpl.Execute(&body, u); err != nil {
		return fmt.Errorf("erro ao renderizar template: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.config.SMTPEmail)
	m.SetHeader("To", u.Email)
	m.SetHeader("Subject", "Uma oferta especial para você!")
	m.SetBody("text/html", body.String())

	return s.dialer.DialAndSend(m)
}

// registerStatus grava o status do envio no banco.
func (s *EmailSender) registerStatus(email, status string) {
	// ▶️ CRIA UMA INSTÂNCIA DO SEU STRUCT `models.EmailStatus`
	record := models.EmailStatus{Email: email, Status: status}
	if err := s.db.Create(&record).Error; err != nil {
		log.Println("Erro ao registrar status:", err)
	}
}

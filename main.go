package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"sync"

	"sendemail/models"

	"gopkg.in/gomail.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("==== Envio de E-mails em Massa ====")

	// ▶️ Lê dados de conexão via terminal
	config := readDBConfigFromTerminal()

	// ▶️ Monta DSN
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
	)

	// ▶️ Conecta no PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar no banco:", err)
	}

	fmt.Println("✅ Conexão ao banco realizada.")

	// ▶️ Executa envio de e-mails
	sendEmails(db)

	// ▶️ Pause antes de sair
	fmt.Println("Processo finalizado. Pressione ENTER para sair...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// readDBConfigFromTerminal lê dados do banco via terminal
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func readDBConfigFromTerminal() DBConfig {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Host do banco: ")
	host, _ := reader.ReadString('\n')

	fmt.Print("Porta do banco: ")
	port, _ := reader.ReadString('\n')

	fmt.Print("Usuário do banco: ")
	user, _ := reader.ReadString('\n')

	fmt.Print("Senha do banco: ")
	password, _ := reader.ReadString('\n')

	fmt.Print("Nome do banco: ")
	dbname, _ := reader.ReadString('\n')

	return DBConfig{
		Host:     strings.TrimSpace(host),
		Port:     strings.TrimSpace(port),
		User:     strings.TrimSpace(user),
		Password: strings.TrimSpace(password),
		DBName:   strings.TrimSpace(dbname),
	}
}

// sendEmails executa envio de e-mails
func sendEmails(db *gorm.DB) {
	models.AutoMigrate(db)

	// ▶️ Buscar usuários
	var usuarios []models.Usuario
	if err := db.Find(&usuarios).Error; err != nil {
		log.Println("Erro ao buscar usuários:", err)
		return
	}

	if len(usuarios) == 0 {
		log.Println("Nenhum usuário encontrado para envio.")
		return
	}

	// ▶️ Carregar template
	tmpl, err := template.ParseFiles("templates/email.html")
	if err != nil {
		log.Println("Erro ao carregar template:", err)
		return
	}

	// ▶️ Config SMTP
	emailFrom := "seuemail@gmail.com"
	emailPassword := "APP_PASSWORD"
	smtpHost := "smtp.gmail.com"
	smtpPort := 587

	numWorkers := 5
	dispatchEmails(usuarios, tmpl, db, emailFrom, emailPassword, smtpHost, smtpPort, numWorkers)
}

// dispatchEmails cria pool de workers
func dispatchEmails(usuarios []models.Usuario, tmpl *template.Template, db *gorm.DB, emailFrom, emailPassword, smtpHost string, smtpPort, numWorkers int) {
	jobs := make(chan models.Usuario, len(usuarios))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go emailWorker(w, jobs, tmpl, db, emailFrom, emailPassword, smtpHost, smtpPort, &wg)
	}

	for _, u := range usuarios {
		jobs <- u
	}
	close(jobs)

	wg.Wait()
}

// emailWorker processa envios
func emailWorker(id int, jobs <-chan models.Usuario, tmpl *template.Template, db *gorm.DB, emailFrom, emailPassword, smtpHost string, smtpPort int, wg *sync.WaitGroup) {
	defer wg.Done()

	for u := range jobs {
		if err := sendEmail(u, tmpl, emailFrom, emailPassword, smtpHost, smtpPort); err != nil {
			log.Printf("[Worker %d] Erro ao enviar para %s: %v", id, u.Email, err)
			registerStatus(db, u.Email, "erro")
		} else {
			log.Printf("[Worker %d] E-mail enviado para %s", id, u.Email)
			registerStatus(db, u.Email, "sucesso")
		}
	}
}

// sendEmail envia e-mail
func sendEmail(u models.Usuario, tmpl *template.Template, emailFrom, emailPassword, smtpHost string, smtpPort int) error {
	var body bytes.Buffer
	if err := tmpl.Execute(&body, u); err != nil {
		return fmt.Errorf("erro ao renderizar template: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", emailFrom)
	m.SetHeader("To", u.Email)
	m.SetHeader("Subject", "Notificação em Massa")
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer(smtpHost, smtpPort, emailFrom, emailPassword)
	return d.DialAndSend(m)
}

// registerStatus grava status no banco
func registerStatus(db *gorm.DB, email, status string) {
	record := models.EmailStatus{
		Email:  email,
		Status: status,
	}
	if err := db.Create(&record).Error; err != nil {
		log.Println("Erro ao registrar status:", err)
	}
}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// table account
type Account struct {
	Id           int
	Nama         string
	Email        string
	PasswordHash string
	Roles        string
	Nik          int
	NoHp         string
	Saldo        float64
	NoRekening   string
	CreatedAt    time.Time
	UpdateAt     time.Time
}
type AccountReq struct {
	Nama         string  `json:"nama"`
	Email        string  `json:"email"`
	PasswordHash string  `json:"password_hash"`
	Nik          int64   `json:"nik"`
	NoHp         string  `json:"no_hp"`
	Roles        string  `json:"roles"`
	Saldo        float64 `json:"saldo"`
	NoRekening   string  `json:"no_rekening"`
}

type AccountTransactionReq struct {
	NoRekening string `json:"no_rekening"`
	Saldo      int    `json:"saldo"`
}
type Transaction struct {
	Id              int
	AccountId       int
	CodeTransaction string
	TotalAmount     float64
	Status          string
	NoRekeningTo    string
	Remark          string
	CreatedAt       time.Time
	UpdateAt        time.Time
}

var DB *sql.DB

func ConnectDB() {
	// kita buat dari awal

	var err error

	host := "localhost"
	port := "5432"
	user := "postgres"
	password := "1234"
	dbname := "testing_db"

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Failed to connected to database: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Successfully connected to database")

}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func main() {

	router := echo.New()

	// connection database

	ConnectDB()
	defer CloseDB()

	// router.GET("/testing-api", func(c echo.Context) error {
	// 	return c.String(200, "API RUNNING")
	// })

	router.POST("/create-account", CreateAcoount)
	router.GET("/get-account", GetAccount)
	router.GET("/get-saldo/:no_rekening", GetSaldo)
	router.POST("/tabung", TopupSaldo)
	router.POST("/tarik", TarikSaldo)

	router.Logger.Fatal(router.Start(":8080"))

}

func ValidateNumberString(input string) bool {
	// Define regex pattern to match only digits
	re := regexp.MustCompile(`^\d+$`)
	return re.MatchString(input)
}

func Generate10DigitNumberRek() int {
	// rand.Seed(time.Now().UnixNano())
	number := rand.Int63n(1e10) // Angka maksimum 10^10 - 1
	return int(number)
}

// CreateAcoount
func CreateAcoount(c echo.Context) error {

	var accountReq AccountReq
	err := c.Bind(&accountReq)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}

	log.Println("Cheking log accountreq :", accountReq)

	if accountReq.PasswordHash == "" || accountReq.Nama == "" || accountReq.Email == "" || accountReq.Nik == 0 || accountReq.NoHp == "" {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}

	// generate account noRekening
	randomNumber := Generate10DigitNumberRek()
	// int to string
	noRekening := strconv.Itoa(randomNumber)

	// hashing password
	passwordNew, err := bcrypt.GenerateFromPassword([]byte(accountReq.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "Failed to register nasabah",
			"error":   err.Error(),
		})
	}

	accountReq.PasswordHash = string(passwordNew)
	accountReq.Roles = "USER"

	// insert data to database

	query := "INSERT INTO account (nama, email, password_hash,nik, no_hp, roles, saldo, no_rekening, created_at, update_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	_, err = DB.Exec(query, accountReq.Nama, accountReq.Email, accountReq.PasswordHash, accountReq.Nik, accountReq.NoHp, accountReq.Roles, accountReq.Saldo, noRekening, time.Now(), time.Now())
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "Failed to register nasabah",
			"error":   err.Error(),
		})
	}

	return c.JSON(201, map[string]interface{}{
		"message":     "Account created successfully",
		"no-rekening": noRekening,
	})

}

// GetAccount
func GetAccount(c echo.Context) error {

	// cheking get all account
	rows, err := DB.Query("SELECT id, nama, email, roles, nik, no_hp, saldo, no_rekening, created_at, update_at FROM account")
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "Failed to get account",
			"error":   err.Error(),
		})
	}

	defer rows.Close()

	var account []Account

	for rows.Next() {
		var acc Account
		if err := rows.Scan(&acc.Id, &acc.Nama, &acc.Email, &acc.Roles, &acc.Nik, &acc.NoHp, &acc.Saldo, &acc.NoRekening, &acc.CreatedAt, &acc.UpdateAt); err != nil {
			return c.JSON(500, map[string]interface{}{
				"message": "Failed to get account",
				"error":   err.Error(),
			})
		}
		account = append(account, acc)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Success get all account",
		"account": account,
	})
}

// get saldo
func GetSaldo(c echo.Context) error {

	noRekening := c.Param("no_rekening")
	row := DB.QueryRow("SELECT id, nama, email, roles, nik, no_hp, saldo, no_rekening, created_at, update_at FROM account WHERE no_rekening = $1", noRekening)

	var dataId Account
	err := row.Scan(&dataId.Id, &dataId.Nama, &dataId.Email, &dataId.Roles, &dataId.Nik, &dataId.NoHp, &dataId.Saldo, &dataId.NoRekening, &dataId.CreatedAt, &dataId.UpdateAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed Get saldo",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Success get saldo",
		"account": dataId,
	})
}

func generateRandomCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomCode := make([]byte, 5)
	for i := range randomCode {
		randomCode[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(randomCode)
}

func generateTransaction() string {
	// Dapatkan random code
	randomCode := generateRandomCode()

	// Dapatkan tanggal saat ini
	currentTime := time.Now()

	// Format tanggal menjadi "DDMMYY"
	formattedDate := currentTime.Format("020106")

	// Gabungkan format TRX - RANDOMCODE - DDMMYY
	transactionCode := fmt.Sprintf("TRX-%s-%s", randomCode, formattedDate)

	return transactionCode
}

// TopupSaldo
func TopupSaldo(c echo.Context) error {

	var accountReq AccountReq
	err := c.Bind(&accountReq)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}

	// get value saldo from database
	row := DB.QueryRow("SELECT saldo, no_rekening, id  FROM account WHERE no_rekening = $1", accountReq.NoRekening)
	var dataAccount Account
	err = row.Scan(&dataAccount.Saldo, &dataAccount.NoRekening, &dataAccount.Id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed Get saldo",
			"error":   err.Error(),
		})
	}

	if accountReq.Saldo <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": "saldo tidak boleh 0",
		})
	}

	// tambahkan saldo
	newSaldo := dataAccount.Saldo + accountReq.Saldo
	transactionRandomcode := generateTransaction()

	// update saldo to database
	_, err = DB.Exec("UPDATE account SET saldo = $1 WHERE no_rekening = $2", newSaldo, accountReq.NoRekening)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed update saldo",
			"error":   err.Error(),
		})
	}

	// set to save table transaction
	transactionUpdate := Transaction{
		AccountId:       dataAccount.Id,
		CodeTransaction: transactionRandomcode,
		TotalAmount:     accountReq.Saldo,
		Status:          "success",
		Remark:          "TABUNG",
		NoRekeningTo:    accountReq.NoRekening,
		CreatedAt:       time.Now(),
		UpdateAt:        time.Now(),
	}

	// insert to table transaction

	_, err = DB.Exec("INSERT INTO transaction (account_id, code_transaction, total_amount, status, no_rekening_to, remark, created_at, update_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		transactionUpdate.AccountId, transactionUpdate.CodeTransaction, transactionUpdate.TotalAmount, transactionUpdate.Status, transactionUpdate.NoRekeningTo, transactionUpdate.Remark, transactionUpdate.CreatedAt, transactionUpdate.UpdateAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed insert to table transaction",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Success top up saldo",
		"saldo":   newSaldo,
	})
}

// TarikSaldo
func TarikSaldo(c echo.Context) error {

	var accountReq AccountReq
	err := c.Bind(&accountReq)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}

	// get value saldo from database
	row := DB.QueryRow("SELECT saldo, no_rekening, id FROM account WHERE no_rekening = $1", accountReq.NoRekening)
	var dataAccount Account
	err = row.Scan(&dataAccount.Saldo, &dataAccount.NoRekening, &dataAccount.Id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed Get saldo",
			"error":   err.Error(),
		})
	}

	if accountReq.Saldo > dataAccount.Saldo {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": "Saldo tidak cukup",
		})
	}

	// tarik saldo
	newSaldo := dataAccount.Saldo - accountReq.Saldo

	// update saldo to database
	_, err = DB.Exec("UPDATE account SET saldo = $1 WHERE no_rekening = $2", newSaldo, accountReq.NoRekening)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed update saldo",
			"error":   err.Error(),
		})
	}

	// generate random code transaction
	transactionRandomcode := generateTransaction()

	// set to save table transaction
	transactionUpdate := Transaction{
		AccountId:       dataAccount.Id,
		CodeTransaction: transactionRandomcode,
		TotalAmount:     accountReq.Saldo,
		Status:          "success",
		Remark:          "TABUNG",
		NoRekeningTo:    accountReq.NoRekening,
		CreatedAt:       time.Now(),
		UpdateAt:        time.Now(),
	}

	// insert to table transaction

	_, err = DB.Exec("INSERT INTO transaction (account_id, code_transaction, total_amount, status, no_rekening_to, remark, created_at, update_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		transactionUpdate.AccountId, transactionUpdate.CodeTransaction, transactionUpdate.TotalAmount, transactionUpdate.Status, transactionUpdate.NoRekeningTo, transactionUpdate.Remark, transactionUpdate.CreatedAt, transactionUpdate.UpdateAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed insert to table transaction",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Success tarik saldo",
		"saldo":   newSaldo,
	})
}

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Datos de conexión
	user := "..."
	password := "..."
	host := "..."
	port := "..."
	dbname := "parque_atracciones"

	// Crear cadena de conexión
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbname)

	// Abrir la conexión a la base de datos
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error al abrir la base de datos: %v\n", err)
	}
	defer db.Close()

	// Probar la conexión
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error al conectar con la base de datos: %v\n", err)
	}
	fmt.Println("Conexión exitosa a la base de datos")

	// Ejemplo de consulta: obtener la versión de MySQL
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		log.Fatalf("Error al ejecutar la consulta: %v\n", err)
	}
	fmt.Printf("Versión de MySQL: %s\n", version)
}

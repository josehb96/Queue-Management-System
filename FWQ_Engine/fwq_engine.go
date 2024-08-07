package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
)

/**
* Estructura del visitante
* https://devjaime.medium.com/mi-primer-api-en-golang-9461996753dc
 */
type visitante struct {
	ID           string `json:"id"`
	Nombre       string `json:"nombre"`
	Password     string `json:"contraseña"`
	Posicionx    int    `json:"posicionx"`
	Posiciony    int    `json:"posiciony"`
	Destinox     int    `json:"destinox"`
	Destinoy     int    `json:"destinoy"`
	DentroParque int    `json:"dentroParque"`
	IdEnParque   string `json:"idEnParque"`
	UltimoEvento string `json:"ultimoEvento"`
	Parque       string `json:"parqueAtracciones"`
}

/**
* Estructura de las atracciones
 */
type atraccion struct {
	ID           string `json:"id"`
	TCiclo       int    `json:"tciclo"`
	NVisitantes  int    `json:"nvisitantes"`
	Posicionx    int    `json:"posicionx"`
	Posiciony    int    `json:"posiciony"`
	TiempoEspera int    `json:"tiempoEspera"`
	Estado       string `json:"estado"`
	Parque       string `json:"parqueAtracciones"`
}

/**
* Estructura del objeto coord que almacenan las coordenadas de la cuidad
 */
type coord struct {
	Lon float32 `json: "lon"`
	Lat float32 `json: "lat"`
}

/**
* Estructura que almacena la información de la temperatura
 */
type weather struct {
	Id          int    `json: "id"`
	Main        string `json: "main"`
	Description string `json: "description"`
	Icon        string `json: "icon"`
}

/**
* Estructura que almacena la información de la temperatura de la cuidad
 */
type temperatura struct {
	Temp       float32 `json:"temp"`
	Feels_like float32 `json:"feels_like"`
	Temp_min   float32 `json:"temp_min"`
	Temp_max   float32 `json:"temp_max"`
	Pressure   float32 `json: "pressure"`
	Humidity   float32 `json:"humidity"`
}

/**
* Estructura que almacena la información del viento
 */
type wind struct {
	Speed float32 `json:"speed"`
	Deg   float32 `json: "deg"`
}

/**
* Estructura que almacena la candiad de nubes que hay
 */
type clouds struct {
	All float32 `json:"all"`
}

/**
* Estructura que almacena información de la cuidad
 */
type sys struct {
	Tipo    int     `json:"tipo"`
	Id      float32 `json:"id"`
	Country string  `json:"country"`
	Sunrise float32 `json:"sunrise"`
	Sunset  float32 `json:"sunset"`
}

/**
* Estructura que almacena la información de la cuidad y su temperatura
 */
/*type ciudad struct {
	Coordenadas       coord       `json:"coordenadas"`
	Tiempo            weather     `json:"tiempo"`
	Base              string      `json:"base"`
	Temperaturaciudad temperatura `json:"temperaturaciudad"`
	Visibility        float32     `json:"visibility"`
	Viento            wind        `json:"viento"`
	Nubes             clouds      `json:"nubes"`
	Dt                float32     `json:"dt"`
	Informacionciudad sys         `json:"informacionciudad"`
	Timezone          float32     `json:"timezone"`
	Id                float32     `json:"id"`
	Name              string      `json:"name"`
	Cod               float32     `json:"cod"`
}*/

type ciudad struct {
	Cuadrante   string  `json:"cuadrante"`
	Nombre      string  `json:"name"`
	Temperatura float32 `json:"temp"`
}

// Array de bytes aleatorios para la implementación de seguridad del kafka
var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func main() {

	IpKafka := os.Args[1]
	PuertoKafka := os.Args[2]
	numeroVisitantes := os.Args[3]
	ciudadesElegidas := os.Args[4]
	IpFWQWaiting := os.Args[5]
	PuertoWaiting := os.Args[6]

	fmt.Println("Creado un engine que atiende peticiones por " + IpKafka + ":" + PuertoKafka + ", limita el parque a " + numeroVisitantes + " visitantes y manda peticiones a un servidor de tiempos de espera situado en " + IpFWQWaiting + ":" + PuertoWaiting + ".\n")

	//Creamos el topic...Cambiar la Ipkafka en la función principal
	//Si no se ejecuta el programa, se cierra el kafka?
	crearTopics(IpKafka, PuertoKafka, "peticiones")
	crearTopics(IpKafka, PuertoKafka, "respuesta-login")
	crearTopics(IpKafka, PuertoKafka, "movimiento-mapa")

	// SECURIZAMOS LA COMUNICACIÓN EN KAFKA
	// Cargamos la clave de cifrado AES del archivo
	fichero, err := ioutil.ReadFile("claveCifradoAES.txt")
	if err != nil {
		log.Fatal("Error al leer el archivo de la clave de cifrado AES: ", err)
	}

	clave := string(fichero) // Clave de 32 bits

	// Visitantes, atracciones que se encuentran en la BD
	var visitantesRegistrados []visitante
	var conn = conexionBD()
	maxVisitantes, _ := strconv.Atoi(numeroVisitantes)
	establecerMaxVisitantes(conn, maxVisitantes)

	go consumidorEngine(IpKafka, PuertoKafka, maxVisitantes, clave)

	// Guardamos el mapa del parque en curso cada cierto tiempo
	go guardarMapaBD()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			mensaje := "Engine no disponible"
			mensajeCifrado, err := encriptacionAES(string(mensaje), clave)
			if err != nil {
				panic(err)
			}
			mensajeJson, err := json.Marshal(mensajeCifrado)
			if err != nil {
				fmt.Printf("Error a la hora de codificar el mensaje: %v\n", err)
			}

			productorMapa(IpKafka, PuertoKafka, mensajeJson)

			fmt.Println()
			fmt.Println("Engine apagado manualmente")
			pprof.StopCPUProfile()
			os.Exit(1)
		}
	}()

	for {
		visitantesRegistrados, _ = obtenerVisitantesBD(conn) // Obtenemos los visitantes registrados actualmente
		fmt.Println("*********** FUN WITH QUEUES RESORT ACTIVITY MAP *********")
		fmt.Println("ID   	" + "		Nombre      " + "	Pos.      " + "	Destino      " + "	DentroParque")
		//Hay que usar la función TrimSpace porque al parecer tras la obtención de valores de BD se agrega un retorno de carro a cada variable
		//Mostramos los visitantes registrados en la aplicación actualmente
		for i := 0; i < len(visitantesRegistrados); i++ {
			fmt.Println(strings.TrimSpace(visitantesRegistrados[i].ID) + " 		" + strings.TrimSpace(visitantesRegistrados[i].Nombre) +
				"    " + "	(" + strings.TrimSpace(strconv.Itoa(visitantesRegistrados[i].Posicionx)) + "," + strings.TrimSpace(strconv.Itoa(visitantesRegistrados[i].Posiciony)) +
				")" + "	    " + "    (" + strings.TrimSpace(strconv.Itoa(visitantesRegistrados[i].Destinox)) + "," + strings.TrimSpace(strconv.Itoa(visitantesRegistrados[i].Destinoy)) +
				")" + "	             " + strings.TrimSpace(strconv.Itoa(visitantesRegistrados[i].DentroParque)))
		}

		fmt.Println() // Para mejorar la visualización

		// Cada X segundos se conectará al servidor de tiempos para actualizar los tiempos de espera de las atracciones
		time.Sleep(time.Duration(5 * time.Second))
		atracciones, _ := obtenerAtraccionesBD(conn) // Obtenemos las atracciones actualizadas
		conexionTiempoEspera(conn, IpFWQWaiting, PuertoWaiting, atracciones)

		actualizarClimaParque(ciudadesElegidas, atracciones)

		fmt.Println() // Para mejorar la visualización

	}

}

func guardarMapaBD() {

	//Accediendo a la base de datos
	//Abrimos la conexion con la base de datos
	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/parque_atracciones")
	//Si la conexión falla mostrara este error
	if err != nil {
		panic(err.Error())
	}
	//Cierra la conexion con la bd
	defer db.Close()

	for {

		var mapa [20][20]string

		visitantes, _ := obtenerVisitantesParque(db) // Obtenemos los visitantes del parque actualizados
		atracciones, _ := obtenerAtraccionesBD(db)   // Obtenemos las atracciones actualizadas
		mapaActualizado := asignacionPosiciones(visitantes, atracciones, mapa)

		var filaParque string

		for i := 0; i < len(mapaActualizado); i++ {
			for j := 0; j < len(mapaActualizado); j++ {
				if j == 19 { // Si ya estamos en la última columna

					filaParque = filaParque + mapaActualizado[i][j]

					// Preparamos para prevenir inyecciones SQL
					sentenciaPreparada, err := db.Prepare("UPDATE mapa SET infoParque = ? WHERE fila = ?")
					if err != nil {
						panic("Error al preparar la sentencia de modificación del mapa: " + err.Error())
					}

					// Ejecutar sentencia, un valor por cada '?'
					_, err = sentenciaPreparada.Exec(filaParque, i+1)
					if err != nil {
						panic("Error al modificar la posición del visitante en la BD: " + err.Error())
					}

					sentenciaPreparada.Close()

					filaParque = "" // Reiniciamos la cadena

				} else {
					filaParque = filaParque + mapaActualizado[i][j]
				}
			}

		}

		time.Sleep(1 * time.Second) // Guardamos el mapa cada segundo en la BD
	}

}

/* Función que comprueba si el aforo del parque está completo o no */
func parqueLleno(db *sql.DB, maxAforo int) bool {

	var lleno bool = false

	// Comprobamos si las credenciales de acceso son válidas
	results, err := db.Query("SELECT * FROM visitante WHERE dentroParque = 1")

	if err != nil {
		fmt.Println("Error al hacer la consulta sobre la BD para comprobar el aforo: " + err.Error())
	}

	visitantesDentroParque := 0 // Variable en la que vamos a almacenar el número de visitantes que se encuentran en el parque

	// Vamos recorriendo las filas devueltas para obtener el nº de visitanes dentro del parque
	for results.Next() {
		visitantesDentroParque++
	}

	results.Close() // Cerramos la conexión a la BD

	// Si el aforo está al completo
	if visitantesDentroParque >= maxAforo {
		lleno = true
	}

	return lleno

}

/* Función que permite eliminar un elemento de un slice */
/*func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}*/

/* Función que nos sirve para comprobar el hash de un password */
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

/* Función que nos simplifica la llamada de la función EncodeToString en base 64 */
func encodeBase64(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

/* Función que nos simplifica la llamada a DecodeString en base 64 */
func decodeBase64(s string) []byte {
	datos, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return datos
}

/* Función de encriptación que utiliza el algoritmo AES */
func encriptacionAES(texto, claveSecreta string) (string, error) {

	bloqueDeCifrado, err := aes.NewCipher([]byte(claveSecreta)) // Creamos un nuevo bloque de cifrado AES
	if err != nil {
		return "", err
	}

	textoPlano := []byte(texto)
	cfb := cipher.NewCFBEncrypter(bloqueDeCifrado, iv) // Creamos el stream de encriptación
	textoCifrado := make([]byte, len(textoPlano))
	cfb.XORKeyStream(textoCifrado, textoPlano) // Sustituye cada byte de textoPlano por cada byte en el stream de bytes cifrado (textoCifrado)

	return encodeBase64(textoCifrado), nil

}

/* Función de desencriptación que utiliza el algoritmo AES */
func desencriptacionAES(texto, claveSecreta string) (string, error) {

	bloqueDeCifrado, err := aes.NewCipher([]byte(claveSecreta)) // Creamos un nuevo bloque de cifrado AES
	if err != nil {
		return "", err
	}

	textoCifrado := decodeBase64(texto)
	cfb := cipher.NewCFBDecrypter(bloqueDeCifrado, iv) // Creamos el stream de desencriptación
	textoPlano := make([]byte, len(textoCifrado))
	cfb.XORKeyStream(textoPlano, textoCifrado) // Sustituye cada byte de textoCifrado por cada byte en textoPlano

	return string(textoPlano), nil

}

/* Función que almacena los registros de auditoría en la tabla visitante */
func RegistroLog(db *sql.DB, ipPuerto, idVisitante, accion, descripcion string) {

	// Añadimos el evento de log de error al visitante
	sentenciaPreparada, err := db.Prepare("UPDATE visitante SET ultimoEvento = ? WHERE id = ?")
	if err != nil {
		panic("Error al preparar la sentencia de modificación del log: " + err.Error())
	}

	defer sentenciaPreparada.Close()

	var eventoLog string // Variable donde vamos a guardar la información de log que le vamos a pasar a la BD

	dateTime := time.Now().Format("2006-01-02 15:04:05") // Fecha y hora del evento
	ipVisitante := ipPuerto                              // IP y puerto de quién ha provocado el evento
	accionRealizada := accion                            // Que acción se realiza
	descripcionEvento := descripcion                     // Parámetros o descripción del evento

	eventoLog += dateTime + " | "
	eventoLog += ipVisitante + " | "
	eventoLog += accionRealizada + " | "
	eventoLog += descripcionEvento

	_, err = sentenciaPreparada.Exec(eventoLog, idVisitante)
	if err != nil {
		panic("Error al registrar el evento de log: " + err.Error())
	}

}

/* Función que recibe del gestor de colas las credenciales de los visitantes que quieren iniciar sesión para entrar en el parque */
func consumidorEngine(IpKafka, PuertoKafka string, maxVisitantes int, clave string) {

	//Accediendo a la base de datos
	//Abrimos la conexion con la base de datos
	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/parque_atracciones")
	//Si la conexión falla mostrara este error
	if err != nil {
		panic(err.Error())
	}
	//Cierra la conexion con la bd
	defer db.Close()

	direccionKafka := IpKafka + ":" + PuertoKafka
	//Configuración de lector de kafka
	conf := kafka.ReaderConfig{
		Brokers:     []string{direccionKafka},
		Topic:       "peticiones", //Topico que hemos creado
		GroupID:     "visitantes",
		StartOffset: kafka.LastOffset,
	}

	reader := kafka.NewReader(conf)

	for {

		m, err := reader.ReadMessage(context.Background())

		if err != nil {
			fmt.Println("Ha ocurrido algún error a la hora de conectarse con el kafka", err)
		}

		cadena, err := desencriptacionAES(string(m.Value), clave)
		if err != nil {
			panic(err)
		}

		fmt.Println("Petición recibida: " + cadena)

		cadenaPeticion := strings.Split(cadena, ":")

		alias := cadenaPeticion[0]
		peticion := cadenaPeticion[1]
		destino := strings.Split(cadenaPeticion[2], ",")
		destinoX, _ := strconv.Atoi(strings.TrimSpace(destino[0]))
		destinoY, _ := strconv.Atoi(strings.TrimSpace(destino[1]))

		v := visitante{
			ID:       strings.TrimSpace(alias),
			Password: strings.TrimSpace(peticion),
			Destinox: destinoX,
			Destinoy: destinoY,
		}

		// Nos guardamos la posible contraseña recibida
		var contraseña string = v.Password

		// Obtenemos el hash de la contraseña del visitante en caso de que el ID coincida con alguno almacenado en la BD
		results := db.QueryRow("SELECT contraseña FROM visitante WHERE id = ?", v.ID)

		var hash string

		results.Scan(&hash)

		var respuesta string = ""

		// Si las credenciales coinciden con las de un visitante registrado en la BD y el parque no está lleno
		if CheckPasswordHash(contraseña, hash) && !parqueLleno(db, maxVisitantes) {

			// Actualizamos el estado del visitante en la BD
			sentenciaPreparada, err := db.Prepare("UPDATE visitante SET dentroParque = 1, destinox = ?, destinoy = ? WHERE id = ?")
			if err != nil {
				panic("Error al preparar la sentencia de modificación: " + err.Error())
			}

			// Ejecutar sentencia, un valor por cada '?'
			_, err = sentenciaPreparada.Exec(v.Destinox, v.Destinoy, v.ID)
			if err != nil {
				panic("Error al actualizar el estado del visitante respecto al parque: " + err.Error())
			}

			// Nos guardamos los visitantes del parque asociados a este engine
			//visitantesDelEngine = append(visitantesDelEngine, v.ID)

			respuesta += alias + ":" + "Acceso concedido"
			respuestaCifrada, err := encriptacionAES(respuesta, clave)
			if err != nil {
				panic(err)
			}
			productorLogin(IpKafka, PuertoKafka, respuestaCifrada)

			sentenciaPreparada.Close()

			// Si se nos ha mandado un movimiento
		} else if peticion == "IN" || peticion == "N" || peticion == "S" || peticion == "W" || peticion == "E" || peticion == "NW" ||
			peticion == "NE" || peticion == "SW" || peticion == "SE" {

			// Comprobamos que el alias pertenezca a un visitante que se encuentra en el parque
			results, err := db.Query("SELECT * FROM visitante WHERE id = ?", v.ID)

			if err != nil {
				fmt.Println("Error al hacer la consulta sobre la BD para el login: " + err.Error())
			}

			// Actualizamos el destino del visitante en la BD
			sentenciaPreparada, err := db.Prepare("UPDATE visitante SET destinox = ?, destinoy = ? WHERE id = ?")
			if err != nil {
				panic("Error al preparar la sentencia de modificación de destino: " + err.Error())
			}

			// Ejecutar sentencia, un valor por cada '?'
			_, err = sentenciaPreparada.Exec(v.Destinox, v.Destinoy, v.ID)
			if err != nil {
				panic("Error al actualizar el destino del visitante: " + err.Error())
			}

			sentenciaPreparada.Close()

			if results.Next() {

				var mapa [20][20]string
				visitantesParque, _ := obtenerVisitantesParque(db)             // Obtenemos los visitantes del parque actualizados
				mueveVisitante(db, alias, peticion, visitantesParque)          // Movemos al visitante en base al movimiento recibido
				visitantesParqueActualizados, _ := obtenerVisitantesParque(db) // Obtenemos los visitantes del parque actualizados
				// Preparamos el mapa a enviar a los visitantes que se encuentran en el parque
				atracciones, _ := obtenerAtraccionesBD(db) // Obtenemos las atracciones actualizadas
				mapaActualizado := asignacionPosiciones(visitantesParqueActualizados, atracciones, mapa)
				var representacion string
				for i := 0; i < len(mapaActualizado); i++ {
					for j := 0; j < len(mapaActualizado); j++ {
						if j == 19 {
							representacion = representacion + mapaActualizado[i][j] + "\n"
						} else {
							representacion = representacion + mapaActualizado[i][j]
						}
					}
				}

				// Encriptamos el mapa
				representacionCifrada, err := encriptacionAES(representacion, clave)
				if err != nil {
					panic(err)
				}

				//Convertimos el mapaActualizado a formato jSON
				//Esta función devuelve un array de byte
				mapaJson, err := json.Marshal(representacionCifrada)
				//En formato json tiene encuenta el salto de linea por lo que hay que ver si al decodificarlo se quita
				if err != nil {
					fmt.Printf("Error a la hora de codificar el mapa: %v\n", err)
				}

				productorMapa(IpKafka, PuertoKafka, mapaJson) // Mandamos el mapa actualizado a los visitantes que se encuentran en el parque
				results.Close()

			} else { // Si el alias no pertenece a un visitante del parque
				respuesta += alias + ":" + "Parque cerrado"
				respuestaCifrada, err := encriptacionAES(respuesta, clave)
				if err != nil {
					panic(err)
				}
				productorLogin(IpKafka, PuertoKafka, respuestaCifrada)
				results.Close()
			}
			// Si se nos ha solicitado una salida del parque
		} else if peticion == "OUT" {

			// Sacamos del parque al visitante y reinciamos tanto su posición actual como su destino
			sentenciaPreparada, err := db.Prepare("UPDATE visitante SET dentroParque = 0, posicionx = 0, posiciony = 0, destinox = -1, destinoy = -1 WHERE id = ?")
			if err != nil {
				panic("Error al preparar la sentencia de modificación: " + err.Error())
			}

			// Ejecutar sentencia, un valor por cada '?'
			_, err = sentenciaPreparada.Exec(v.ID)
			if err != nil {
				panic("Error al actualizar el estado del visitante respecto al parque: " + err.Error())
			}

			sentenciaPreparada.Close()

			RegistroLog(db, IpKafka+":"+PuertoKafka, v.ID, "Baja", "El visitante "+v.ID+" ha salido del parque") // Registramos el evento de log

		} else { // Si las credenciales enviadas para iniciar sesión no son válidas

			if parqueLleno(db, maxVisitantes) {
				respuesta += alias + ":" + "Aforo al completo"
				respuestaCifrada, err := encriptacionAES(respuesta, clave)
				if err != nil {
					panic(err)
				}
				productorLogin(IpKafka, PuertoKafka, respuestaCifrada)
			} else {
				respuesta += alias + ":" + "Parque cerrado"
				respuestaCifrada, err := encriptacionAES(respuesta, clave)
				if err != nil {
					panic(err)
				}
				productorLogin(IpKafka, PuertoKafka, respuestaCifrada)
			}
		}

	}

}

/* Función que envía el mensaje de respuesta a la petición de login de un visitante */
func productorLogin(IpBroker, PuertoBroker string, respuesta string) {

	var brokerAddress string = IpBroker + ":" + PuertoBroker
	var topic string = "respuesta-login"

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{brokerAddress},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("Key-Login"),
		Value: []byte(respuesta),
	})

	if err != nil {
		fmt.Println("No se puede mandar el mensaje de respuesta a la petición de login: " + err.Error())
	}

}

/* Función que abre una conexion con la bd */
func conexionBD() *sql.DB {
	//Accediendo a la base de datos
	/*****Flate blod **/
	//Abrimos la conexion con la base de datos
	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/parque_atracciones")
	//Si la conexión falla mostrara este error
	if err != nil {
		panic(err.Error())
	}
	//Cierra la conexion con la bd
	//defer db.Close()
	return db
}

/*
* Función que obtiene todos los visitantes que se encuentran la BD
* @return []visitante : Arrays de los visitantes obtenidos en la sentencia
* @return error : Error en caso de que no se haya podido obtener ninguno
 */
func obtenerVisitantesBD(db *sql.DB) ([]visitante, error) {

	//Ejecutamos la sentencia
	results, err := db.Query("SELECT * FROM visitante")

	if err != nil {
		return nil, err //devolvera nil y error en caso de que no se pueda hacer la consulta
	}

	//Cerramos la base de datos
	defer results.Close()

	//Declaramos el array de visitantes
	var visitantes []visitante

	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {

		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_visitante visitante

		if err := results.Scan(&fwq_visitante.ID, &fwq_visitante.Nombre,
			&fwq_visitante.Password, &fwq_visitante.Posicionx,
			&fwq_visitante.Posiciony, &fwq_visitante.Destinox, &fwq_visitante.Destinoy,
			&fwq_visitante.DentroParque, &fwq_visitante.IdEnParque, &fwq_visitante.UltimoEvento, &fwq_visitante.Parque); err != nil {
			return visitantes, err
		}

		//Vamos añadiendo los visitantes al array
		visitantes = append(visitantes, fwq_visitante)
	}

	if err = results.Err(); err != nil {
		return visitantes, err
	}

	return visitantes, nil

}

/*
* Función que obtiene todos los visitantes que se encuentran en el parque de la BD
* @return []visitante : Arrays de los visitantes obtenidos en la sentencia
* @return error : Error en caso de que no se haya podido obtener ninguno
 */
func obtenerVisitantesParque(db *sql.DB) ([]visitante, error) {

	//Ejecutamos la sentencia
	results, err := db.Query("SELECT * FROM visitante WHERE dentroParque = 1")

	if err != nil {
		return nil, err //devolvera nil y error en caso de que no se pueda hacer la consulta
	}

	//Cerramos la base de datos
	defer results.Close()

	//Declaramos el array de visitantes
	var visitantes []visitante

	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {

		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_visitante visitante

		if err := results.Scan(&fwq_visitante.ID, &fwq_visitante.Nombre,
			&fwq_visitante.Password, &fwq_visitante.Posicionx,
			&fwq_visitante.Posiciony, &fwq_visitante.Destinox, &fwq_visitante.Destinoy,
			&fwq_visitante.DentroParque, &fwq_visitante.IdEnParque, &fwq_visitante.UltimoEvento, &fwq_visitante.Parque); err != nil {
			return visitantes, err
		}

		//Vamos añadiendo los visitantes al array
		visitantes = append(visitantes, fwq_visitante)
	}

	if err = results.Err(); err != nil {
		return visitantes, err
	}

	return visitantes, nil

}

/*
* Función que obtiene las atracciones del parque
* @return []atraccion : Array con las atracciones del parque
* @return error : Error en caso de que no se ha podido obtener las atracciones
 */
func obtenerAtraccionesBD(db *sql.DB) ([]atraccion, error) {

	//Ejecutamos la sentencia
	results, err := db.Query("SELECT * FROM atraccion")

	if err != nil {
		return nil, err //devolvera nil y error en caso de que no se pueda hacer la consulta
	}

	// Nos aseguramos de que se cierre la base de datos
	defer results.Close()

	//Declaramos el array de atracciones
	var atraccionesParque []atraccion

	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {

		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_atraccion atraccion

		if err := results.Scan(&fwq_atraccion.ID, &fwq_atraccion.TCiclo,
			&fwq_atraccion.NVisitantes, &fwq_atraccion.Posicionx,
			&fwq_atraccion.Posiciony, &fwq_atraccion.TiempoEspera,
			&fwq_atraccion.Estado, &fwq_atraccion.Parque); err != nil {
			return atraccionesParque, err
		}

		//Vamos añadiendo las atracciones al array
		atraccionesParque = append(atraccionesParque, fwq_atraccion)

	}

	if err = results.Err(); err != nil {
		return atraccionesParque, err
	}

	return atraccionesParque, nil

}

/*
* Función que forma el mapa del parque conteniendo a los visitantes y las atracciones
* @return [20][20]string : Matriz bidimensional representando el mapa
 */
func asignacionPosiciones(visitantes []visitante, atracciones []atraccion, mapa [20][20]string) [20][20]string {

	//Asignamos los id de los visitantes
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(visitantes); k++ {
				if i == visitantes[k].Posicionx && j == visitantes[k].Posiciony && visitantes[k].DentroParque == 1 {
					mapa[i][j] = visitantes[k].IdEnParque + "|"
				}
			}
		}
	}

	//Asignamos los valores de tiempo de espera de las atracciones
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(atracciones); k++ {
				if i == atracciones[k].Posicionx && j == atracciones[k].Posiciony && atracciones[k].Estado == "Abierta" {
					mapa[i][j] = strconv.Itoa(atracciones[k].TiempoEspera) + "|"
				} else if i == atracciones[k].Posicionx && j == atracciones[k].Posiciony && atracciones[k].Estado == "Cerrada" {
					mapa[i][j] = "(" + strconv.Itoa(atracciones[k].TiempoEspera) + ")" + "|"
				}
			}
		}
	}

	// Las casillas del mapa que no tengan ni visitantes ni atracciones las representamos con una guión
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			if len(mapa[i][j]) == 0 {
				mapa[i][j] = "-" + "|"
			}
		}
	}
	return mapa
}

/*
* Función que se conecta al servidor de tiempos para obtener los tiempos de espera actualizados
 */
func conexionTiempoEspera(db *sql.DB, IpFWQWating, PuertoWaiting string, atracciones []atraccion) {

	fmt.Println() // Por limpieza
	fmt.Println("***Conexión con el servidor de tiempo de espera***")
	fmt.Println() // Por limpieza
	//fmt.Println("Arrancando el engine para atender los tiempos en el puerto: " + IpFWQWating + ":" + PuertoWaiting)
	var connType string = "tcp"
	conn, err := net.Dial(connType, IpFWQWating+":"+PuertoWaiting)

	if err != nil {
		log.Println("ERROR: El servidor de tiempos de espera no está disponible", err.Error())
	} else {

		fmt.Println("***Actualizando los tiempos de espera***")
		fmt.Println() // Por limpieza

		var infoAtracciones string = ""

		for i := 0; i < len(atracciones); i++ {
			infoAtracciones += atracciones[i].ID + ":"
			infoAtracciones += strconv.Itoa(atracciones[i].TCiclo) + ":"
			infoAtracciones += strconv.Itoa(atracciones[i].NVisitantes) + ":"
			infoAtracciones += strconv.Itoa(atracciones[i].TiempoEspera) + "|"
		}

		infoAtracciones += "\n" // Le añadimos el salto de línea porque los sockets los estamos leyendo hasta final de línea

		fmt.Println("Enviando información de las atracciones...")

		conn.Write([]byte(infoAtracciones))                        // Mandamos el id:tiempoCiclo:nºvisitantes de cada atracción en un string
		tiemposEspera, _ := bufio.NewReader(conn).ReadString('\n') // Obtenemos los tiempos de espera actualizados

		if tiemposEspera != "" {

			log.Println("Tiempos de espera actualizados: " + tiemposEspera)

			arrayTiemposEspera := strings.Split(tiemposEspera, "|")

			// Actualizamos los tiempos de espera de las atracciones en la BD
			actualizaTiemposEsperaBD(db, arrayTiemposEspera)

		} else {
			log.Println("Servidor de tiempos no disponible.")
		}

	}

}

/*
* Función que establece el aforo maximo permitido en el parque de atracciones
 */
func establecerMaxVisitantes(db *sql.DB, numero int) {

	//Ejecutamos la sentencia
	results, err := db.Query("SELECT * FROM parque")

	if err != nil {
		panic("Error al hacer la consulta del parque" + err.Error()) //devolvera nil y error en caso de que no se pueda hacer la consulta
	}

	//Cerramos la base de datos
	defer results.Close()

	//Recorremos los resultados obtenidos por la consulta
	if results.Next() {

		//Variable donde guardamos la información de cada una filas de la sentencia
		sentenciaPreparada, err := db.Prepare("UPDATE parque SET aforoMaximo = ? WHERE id = ?")

		if err != nil {
			panic("Error al preparar la sentencia" + err.Error()) //devolvera nil y error en caso de que no se pueda hacer la consulta
		}

		defer sentenciaPreparada.Close()

		_, err = sentenciaPreparada.Exec(numero, "SDpark")

		if err != nil {
			panic("Error al establecer el número máximo de visitantes" + err.Error())
		}
	}
}

/* Función que modifica las posiciones de los visitantes en el parque en base a sus movimientos */
func mueveVisitante(db *sql.DB, id, movimiento string, visitantes []visitante) {

	var nuevaPosicionX int
	var nuevaPosicionY int

	for i := 0; i < len(visitantes); i++ {

		if id == visitantes[i].ID { // Modificamos la posición del visitante recibido por kafka

			switch movimiento {
			case "N":
				visitantes[i].Posicionx--
			case "S":
				visitantes[i].Posicionx++
			case "W":
				visitantes[i].Posiciony--
			case "E":
				visitantes[i].Posiciony++
			case "NW":
				visitantes[i].Posicionx--
				visitantes[i].Posiciony--
			case "NE":
				visitantes[i].Posicionx--
				visitantes[i].Posiciony++
			case "SW":
				visitantes[i].Posicionx++
				visitantes[i].Posiciony--
			case "SE":
				visitantes[i].Posicionx++
				visitantes[i].Posiciony++
			}

			if visitantes[i].Posicionx == -1 {
				visitantes[i].Posicionx = 19
			} else if visitantes[i].Posicionx == 20 {
				visitantes[i].Posicionx = 0
			}

			if visitantes[i].Posiciony == -1 {
				visitantes[i].Posiciony = 19
			} else if visitantes[i].Posiciony == 20 {
				visitantes[i].Posiciony = 0
			}

			nuevaPosicionX = visitantes[i].Posicionx
			nuevaPosicionY = visitantes[i].Posiciony

		}

	}

	// MODIFICAMOS la posición de dicho visitante en la BD
	// Preparamos para prevenir inyecciones SQL
	sentenciaPreparada, err := db.Prepare("UPDATE visitante SET posicionx = ?, posiciony = ? WHERE id = ?")
	if err != nil {
		panic("Error al preparar la sentencia de modificación: " + err.Error())
	}

	defer sentenciaPreparada.Close()

	// Ejecutar sentencia, un valor por cada '?'
	_, err = sentenciaPreparada.Exec(nuevaPosicionX, nuevaPosicionY, id)
	if err != nil {
		panic("Error al modificar la posición del visitante en la BD: " + err.Error())
	}

}

/**
 * Función que envia el mapa a los visitantes
 */
func productorMapa(IpBroker, PuertoBroker string, mapa []byte) {

	var brokerAddress string = IpBroker + ":" + PuertoBroker
	var topic string = "movimiento-mapa"

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{brokerAddress},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("Key-Mapa"), //[]byte(strconv.Itoa(i)),
		Value: []byte(mapa),
	})
	if err != nil {
		panic("No se puede mandar el mapa: " + err.Error())
	}
}

/* Función que actualiza los tiempos de espera de las atracciones en la BD */
func actualizaTiemposEsperaBD(db *sql.DB, tiemposEspera []string) {

	results, err := db.Query("SELECT * FROM atraccion")

	// Comrpobamos que no se produzcan errores al hacer la consulta
	if err != nil {
		panic("Error al hacer la consulta a la BD: " + err.Error())
	}

	defer results.Close() // Nos aseguramos de cerrar*/

	i := 0

	// Recorremos todas las filas de la consulta
	for results.Next() {

		// Preparamos para prevenir inyecciones SQL
		sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET tiempoEspera = ? WHERE id = ?")
		if err != nil {
			panic("Error al preparar la sentencia de modificación: " + err.Error())
		}

		defer sentenciaPreparada.Close()

		infoAtraccion := strings.Split(tiemposEspera[i], ":") // Extraemos el id y el tiempo de espera de la atracción

		idAtraccion := infoAtraccion[0]

		nuevoTiempo, err := strconv.Atoi(infoAtraccion[1])

		if err != nil {
			panic("Error al convertir la cadena con el nuevo tiempo de la atracción")
		}

		// Ejecutar sentencia, un valor por cada '?'
		_, err = sentenciaPreparada.Exec(nuevoTiempo, idAtraccion)
		if err != nil {
			panic("Error al modificar el tiempo de espera de la atracción: " + err.Error())
		}

		i++

	}
}

/*
* Función que crea un topic para el envio de los visitantes
 */
func crearTopics(IpBroker, PuertoBroker, nombre string) {
	/**** IMPORTANTE CAMBIAR*/
	//Broker1 se sustituira en localhost:9092
	var broker1 string = IpBroker + ":" + PuertoBroker
	//el localhost:9092 cambiara y sera pasado por parametro
	conn, err := kafka.Dial("tcp", broker1)
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	controller, err := conn.Controller()

	if err != nil {
		panic(err.Error())
	}
	//Creamos una variable del tipo kafka.Conn
	var controllerConn *kafka.Conn
	//Le damos los valores necesarios para crear el controllerConn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()
	//Configuración del topic mapa-visitantes
	topicConfigs := []kafka.TopicConfig{
		kafka.TopicConfig{
			Topic:             nombre,
			NumPartitions:     10, //Cambiamos el número de particiones
			ReplicationFactor: 1,
		},
	}
	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}
}

/* Función que selecciona las 4 ciudades elegidas */
func seleccionaCiudades(ciudades []string, numCiudadesElegidas string) []string {

	numCiudades := strings.Split(numCiudadesElegidas, ",")

	var nombresCiudades []string

	for i := 0; i < 4; i++ {

		num, _ := strconv.Atoi(numCiudades[i])

		nombresCiudades = append(nombresCiudades, ciudades[num-1])

	}

	return nombresCiudades

}

/* Función que muestra un menú para poder seleccionar las 4 ciudades del listado */
func actualizarClimaParque(numerosCiudadesElegidas string, atracciones []atraccion) {

	// Cargamos la clave de cifrado AES del archivo
	ficheroCiudades, err := ioutil.ReadFile("ciudades.txt")
	if err != nil {
		log.Fatal("No se ha podido leer las ciudades del archivo txt")
	}

	nombresCiudades := strings.Split(string(ficheroCiudades), ",")

	ciudadesElegidas := seleccionaCiudades(nombresCiudades, numerosCiudadesElegidas)

	var ciudades []ciudad

	for i, nombreCiudad := range ciudadesElegidas {

		city := ciudad{}

		switch i {
		case 0:
			city.Cuadrante = "arriba-izquierda"
		case 1:
			city.Cuadrante = "arriba-derecha"
		case 2:
			city.Cuadrante = "abajo-izquierda"
		case 3:
			city.Cuadrante = "abajo-derecha"
		}

		city.Nombre = nombreCiudad
		city.Temperatura = obtenerClimaCiudad(nombreCiudad)

		ciudades = append(ciudades, city)

	}

	fmt.Println()
	fmt.Println("La ciudad del cuadrante arriba-izquierda es:", ciudades[0].Nombre, "y su temperatura es: ", ciudades[0].Temperatura, "ºC")
	fmt.Println("La ciudad del cuadrante arriba-derecha es:", ciudades[1].Nombre, "y su temperatura es: ", ciudades[1].Temperatura, "ºC")
	fmt.Println("La ciudad del cuadrante abajo-izquierda es:", ciudades[2].Nombre, "y su temperatura es: ", ciudades[2].Temperatura, "ºC")
	fmt.Println("La ciudad del cuadrante abajo-derecha es:", ciudades[3].Nombre, "y su temperatura es: ", ciudades[3].Temperatura, "ºC")
	fmt.Println()

	almacenarCiudades(ciudades) // Almacenamos las ciudades en la BD para poder consultar su información desde el front

	// Actualizamos la situación del parque en base a las temperaturas de las ciudades
	// OTRA OPCIÓN ES CREAR UNA VARIABLE GLOBAL CON EL ESTADO ACTUAL DE LAS 4 CIUDADES
	actualizarClimaAtracciones(ciudades, atracciones)

}

/* Función que almacena en la BD la información de las 4 ciudades */
func almacenarCiudades(ciudades []ciudad) {

	//Accediendo a la base de datos
	//Abrimos la conexion con la base de datos
	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/parque_atracciones")
	//Si la conexión falla mostrara este error
	if err != nil {
		panic(err.Error())
	}
	//Cierra la conexion con la bd
	defer db.Close()

	// Comprobamos si existe alguna ciudad almacenada previamente en la BD
	results, err := db.Query("SELECT * FROM ciudades")
	if err != nil {
		fmt.Println("Error al hacer la consulta sobre las ciudades en la BD: " + err.Error())
	}

	defer results.Close()

	// Si ya había 4 almacenadas previamente modificamos
	if results.Next() {

		for i := 0; i < 4; i++ {

			// Actualizamos las ciudades en la BD
			sentenciaPreparada, err := db.Prepare("UPDATE ciudades SET nombre = ?, temperatura = ? WHERE cuadrante = ?")
			if err != nil {
				panic("Error al preparar la sentencia de modificación del estado de las ciudades: " + err.Error())
			}

			defer sentenciaPreparada.Close()

			// Ejecutar sentencia, un valor por cada '?'
			_, err = sentenciaPreparada.Exec(ciudades[i].Nombre, ciudades[i].Temperatura, ciudades[i].Cuadrante)
			if err != nil {
				panic("Error al actualizar las ciudades en la BD: " + err.Error())
			}

		}

	} else { // Sino insertamos las ciudades en la BD

		for i := 0; i < 4; i++ {

			// Preparamos para prevenir inyecciones SQL
			sentenciaPreparada, err := db.Prepare("INSERT INTO ciudades (cuadrante, nombre, temperatura) VALUES(?, ?, ?)")
			if err != nil {
				panic("Error al preparar la sentencia de inserción de las ciudades en la BD: " + err.Error())
			}

			defer sentenciaPreparada.Close()

			// Ejecutar sentencia, un valor por cada '?'
			_, err = sentenciaPreparada.Exec(ciudades[i].Cuadrante, ciudades[i].Nombre, ciudades[i].Temperatura)
			if err != nil {
				panic("Error al insertar las ciudades en la BD: " + err.Error())
			}

		}

	}

}

/* Función que actualiza el estado de las atracciones en base al clima del cuadrante donde se encuentre */
func actualizarClimaAtracciones(ciudades []ciudad, atracciones []atraccion) {

	//Accediendo a la base de datos
	//Abrimos la conexion con la base de datos
	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/parque_atracciones")
	//Si la conexión falla mostrara este error
	if err != nil {
		panic(err.Error())
	}
	//Cierra la conexion con la bd
	defer db.Close()

	for _, atraccion := range atracciones {

		// Si la atraccion se encuentra en el cuadrante arriba-izquierda
		if atraccion.Posicionx >= 0 && atraccion.Posicionx <= 9 && atraccion.Posiciony >= 0 && atraccion.Posiciony <= 9 {

			// Comprobamos el clima de la ciudad asociada
			if ciudades[0].Temperatura < 20.0 || ciudades[0].Temperatura > 30.0 {

				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Cerrada", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}

			} else {
				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Abierta", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}
			}
			// Si la atraccion se encuentra en el cuadrante arriba-derecha
		} else if atraccion.Posicionx >= 0 && atraccion.Posicionx <= 9 && atraccion.Posiciony >= 10 && atraccion.Posiciony <= 19 {

			// Comprobamos el clima de la ciudad asociada
			if ciudades[1].Temperatura < 20.0 || ciudades[1].Temperatura > 30.0 {

				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Cerrada", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}

			} else {
				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Abierta", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}
			}
			// Si la atraccion se encuentra en el cuadrante abajo-izquierda
		} else if atraccion.Posicionx >= 10 && atraccion.Posicionx <= 19 && atraccion.Posiciony >= 0 && atraccion.Posiciony <= 9 {

			// Comprobamos el clima de la ciudad asociada
			if ciudades[2].Temperatura < 20.0 || ciudades[2].Temperatura > 30.0 {

				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Cerrada", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}

			} else {
				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Abierta", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}
			}
			// Si la atracción se encuentra en el cuadrante abajo-derecha
		} else if atraccion.Posicionx >= 10 && atraccion.Posicionx <= 19 && atraccion.Posiciony >= 10 && atraccion.Posiciony <= 19 {

			// Comprobamos el clima de la ciudad asociada
			if ciudades[3].Temperatura < 20.0 || ciudades[3].Temperatura > 30.0 {

				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Cerrada", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}

			} else {
				// Actualizamos el estado de la atracción en la BD
				sentenciaPreparada, err := db.Prepare("UPDATE atraccion SET estado = ? WHERE id = ?")
				if err != nil {
					panic("Error al preparar la sentencia de modificación del estado de la atracción: " + err.Error())
				}

				defer sentenciaPreparada.Close()

				// Ejecutar sentencia, un valor por cada '?'
				_, err = sentenciaPreparada.Exec("Abierta", atraccion.ID)
				if err != nil {
					panic("Error al actualizar el estado de la atracción: " + err.Error())
				}
			}

		}

	}

}

/**
*	Función que nos conecta a la API externa para obtener el tiempo
 */
/*
func obtenerCiudad(w http.ResponseWriter, r *http.Request) {
	var apikey string = "c3d8572d0046f36f0c586caa0e2e1d23"
	//Declaramos las variables que vamos a utilizar para las peticiones
	var coordenadasCiudad coord
	var climaCiudad ciudad
	//Obtenemos todos los parametros pasado a esta función cuando se le llame
	vars := mux.Vars(r)
	//Obtenemos los parametros y los almacenamos en la variable coordenadasCiudad
	coordenadasCiudad.lon = float32(vars["lon"])
	coordenadasCiudad.lat = float32(vars["lat"])

	fmt.Println("coordenadasCiudad")
	//Realizamos la petición a la api de terceros
	peticion, err := http.NewRequest("GET", "https://api.openweathermap.org/data/2.5/weather?lat="+
		coordenadasCiudad.lat+"lon="+coordenadasCiudad.lon+"appid="+apiKey+"lang=es"+"units=metric")
	//Comprobamos que no haya ningun error
	if err != nil {
		log.Fatal("Error al crear la petición: %v", err)
	}
	//Cerramos la petición get
	defer peticion.Body.Close()
	//Agregamos encabezados
	peticion.Header.Add("Content-Type", "application/json")
	//Decodificamos el body de la respuesta y lo almacenamos en el climaCiudad parametro
	body, err := json.NewDecoder(peticion.Body).Decode(&climaCiudad)
	//Comprobamos que no haya ningun error
	if err != nil {
		log.Fatal(err)
	}
	//Imprimimos el cuerpo
	fmt.Println(body)
	fmt.Println(climaCiudad)

}
*/

func obtenerClimaCiudad(nombreCiudad string) float32 {

	clienteHttp := &http.Client{}

	// Cargamos el apikey desde fichero para consumir la api de OpenWeather
	fichero, err := ioutil.ReadFile("apikey.txt")
	if err != nil {
		log.Fatal("Error al leer el archivo de la api key de OpenWeather: ", err)
	}
	var apiKey string = string(fichero)

	peticion, err := http.NewRequest("GET", "https://api.openweathermap.org/data/2.5/weather?q="+nombreCiudad+"&appid="+apiKey+"&lang=es"+"&units=metric", nil)

	if err != nil {
		log.Fatalf("Error creando petición: %v", err)
	}

	peticion.Header.Add("Content-Type", "application/json")

	respuesta, err := clienteHttp.Do(peticion)
	if err != nil {
		// Maneja el error de acuerdo a tu situación
		log.Fatalf("Error haciendo petición: %v", err)
	}

	// No olvides cerrar el cuerpo al terminar
	defer respuesta.Body.Close()

	// Vamos a obtener el cuerpo y lo almacenaremos en la variable
	cuerpoRespuesta, err := ioutil.ReadAll(respuesta.Body)
	//var city ciudad
	//body := json.NewDecoder(peticion.Body).Decode(&city)
	if err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}
	respuestaString := strings.Split(string(cuerpoRespuesta), ",")
	respuestaString = strings.Split(respuestaString[7], ":")
	//fmt.Println(respuestaString[2])

	// Convertimos el string a float32
	value, err := strconv.ParseFloat(respuestaString[2], 32)
	if err != nil {
		panic(err)
	}
	temperatura := float32(value)

	return temperatura

}

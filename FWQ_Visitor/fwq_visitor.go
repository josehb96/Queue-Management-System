package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	hho "crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/segmentio/kafka-go"
)

/*
* Estructura del visitante
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
	IdEnParque   string `json:"idParque"`
	UltimoEvento string `json:"ultimoEvento"`
	Parque       string `json:"parqueAtracciones"`
}

/*
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

const (
	connType = "tcp"
)

var v = visitante{ // Guardamos la información del visitante que nos hace falta
	ID:           "",
	Password:     "",
	Posicionx:    0,
	Posiciony:    0,
	Destinox:     -1,
	Destinoy:     -1,
	DentroParque: 0,
}

var a = atraccion{ // Guardamos la información de la atraccion que nos hace falta
	Posicionx:    -1,
	Posiciony:    -1,
	TiempoEspera: -1,
}

// Ininicializamos un vector con bytes aleatorios
var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

//type HexData []byte

/**
* Función main de los visitantes
**/
func main() {

	//Argumentos iniciales
	IpFWQ_Registry := os.Args[1]
	PuertoRegistrySockets := os.Args[2]
	PuertoRegistryApiRest := os.Args[3]
	IpBroker := os.Args[4]
	PuertoBroker := os.Args[5]
	crearTopic(IpBroker, PuertoBroker, "peticiones")
	crearTopic(IpBroker, PuertoBroker, "respuesta-login")
	crearTopic(IpBroker, PuertoBroker, "movimiento-mapa")

	fmt.Println("Creado un visitante que envía peticiones a un registry por " + IpFWQ_Registry + ":" + PuertoRegistrySockets + "/" + PuertoRegistryApiRest + " y a un engine por " + IpBroker + ":" + PuertoBroker)
	fmt.Println() // Por limpieza

	fmt.Println("**Bienvenido al parque de atracciones**")
	fmt.Println()

	MenuParque(IpFWQ_Registry, PuertoRegistrySockets, PuertoRegistryApiRest, IpBroker, PuertoBroker)

}

/*
* Función que pinta el menu del parque
 */
func MenuParque(IpFWQ_Registry, PuertoRegistrySockets, PuertoRegistryApiRest, IpBroker, PuertoBroker string) {
	var opcion int
	//Guardamos la opcion elegida
	for {
		fmt.Println("***Menu parque de atracciones***")
		fmt.Println("1.Crear perfil")
		fmt.Println("2.Editar perfil")
		fmt.Println("3.Moverse por el parque")
		//fmt.Println("4.Salir del parque")
		fmt.Print("Elige la acción a realizar:")
		fmt.Scanln(&opcion)

		switch os := opcion; os {
		case 1:
			CrearPerfil(IpFWQ_Registry, PuertoRegistrySockets, PuertoRegistryApiRest)
		case 2:
			EditarPerfil(IpFWQ_Registry, PuertoRegistrySockets, PuertoRegistryApiRest)
		case 3:
			EntradaParque(IpBroker, PuertoBroker)
		default:
			fmt.Println("Opción invalida, elige otra opción")
		}
	}
}

/* Función que se conecta al módulo FWQ_Registry para crear un nuevo usuario */
func CrearPerfil(ipRegistry, puertoRegistrySockets, puertoRegistryApiRest string) {

	fmt.Println() // Por limpieza
	fmt.Println("**********Creación de perfil***********")
	fmt.Println() // Por limpieza

	// Damos la posibilidad elegir conexión vía sockets o por API_REST
	fmt.Println("Selecciona el tipo de conexión al registry:")
	fmt.Println("1 -> Sockets")
	fmt.Println("2 -> API_REST")
	fmt.Println() // Por limpieza

	var eleccion int
	fmt.Scan(&eleccion)
	fmt.Println() // Por limpieza

	// Si el usuario elige la conexión por sockets
	if eleccion == 1 {

		cert, err := tls.LoadX509KeyPair("cert/cert.pem", "cert/key.pem")
		if err != nil {
			log.Fatal("Error al cargar los ficheros de certificado y clave asociada: ", err)
		}

		config := tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true, // Para que admita certificados autofirmados
		}

		//conn, err := net.Dial(connType, ipRegistry+":"+puertoRegistry) CONEXIÓN INSEGURA
		conn, err := tls.Dial(connType, ipRegistry+":"+puertoRegistrySockets, &config) // CONEXIÓN SEGURA

		if err != nil {
			fmt.Println("Error al conectarse al Registry:", err.Error())
		} else { // Si el visitante establece conexión con el Registry indicado por parámetro

			defer conn.Close() // Nos aseguramos de cerrar la conexión

			conn.Write([]byte("1" + "\n")) // Le pasamos al Registry la opción elegida por el visitante

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Introduce tu ID:")
			id, _ := reader.ReadString('\n')

			// Nos aseguramos de que no sea válido un id en blanco
			if len(id) > 1 {

				conn.Write([]byte(id))

				fmt.Print("Introduce tu nombre:")
				nombre, _ := reader.ReadString('\n')

				// Nos aseguramos de que no sea válido un nombre en blanco
				if len(nombre) > 1 {

					conn.Write([]byte(nombre))

					fmt.Print("Introduce tu contraseña:")
					password, _ := reader.ReadString('\n')

					// Nos aseguramos de que no sea válida una contraseña en blanco
					if len(password) > 1 {

						conn.Write([]byte(password))

						//Escuchando por el relay el mensaje de respuesta del Registry
						message, _ := bufio.NewReader(conn).ReadString('\n')

						// Comprobamos si el Registry nos devuelve un mensaje de respuesta
						if message != "" {
							log.Print("Respuesta del Registry: ", message)
						} else {
							log.Print("Lo siento, el Registry no está disponible en estos momentos.")
						}

					} else {
						fmt.Println("ERROR: Por favor introduzca una contraseña que no sea vacía.")
					}

				} else {
					fmt.Println("ERROR: Por favor introduzca un nombre que no sea vacío.")
				}

			} else {
				fmt.Println("ERROR: Por favor introduzca un ID que no sea vacío.")
			}

		}
	} else if eleccion == 2 { // Si el usuario elige la conexión por API_REST

		var id string

		fmt.Print("Introduce tu ID:")
		fmt.Scanln(&id)

		// Nos aseguramos de que no sea válido un id en blanco
		if len(id) > 1 {

			var nombre string

			fmt.Print("Introduce tu nombre:")
			fmt.Scanln(&nombre)

			// Nos aseguramos de que no sea válido un nombre en blanco
			if len(nombre) > 1 {

				var password string

				fmt.Print("Introduce tu contraseña:")
				fmt.Scanln(&password)

				// Nos aseguramos de que no sea válida una contraseña en blanco
				if len(password) > 1 {

					v := visitante{
						ID:       id,
						Nombre:   nombre,
						Password: password,
					}
					vComoJson, err := json.Marshal(v)
					if err != nil {
						log.Fatalf("Error codificando visitante como JSON: %v", err)
					}

					// Realizamos la composición de los datos
					datos := strings.NewReader(string(vComoJson))

					tr := &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Para que admita certificados autofirmados
					}
					http := &http.Client{Transport: tr}

					// Ahora realizamos el envío de los datos
					res, err := http.Post("https://"+ipRegistry+":"+puertoRegistryApiRest+"/crear/"+v.ID, "application/json", datos)
					if err != nil {
						fmt.Printf("Error al realizar la petición al servidor API REST: %v\n", err)
						return
					}

					// Nos aseguramos de que se cierra el body
					defer res.Body.Close()

					// Realizamos la lectura del body
					body, err := ioutil.ReadAll(res.Body)
					if err != nil {
						fmt.Printf("Error al leer la respuesta recibida: %v\n", err)
						return
					}

					fmt.Println() // Por limpieza
					fmt.Printf("%s", body)
					fmt.Println() // Por limpieza

				} else {
					fmt.Println("ERROR: Por favor introduzca una contraseña que no sea vacía.")
				}

			} else {
				fmt.Println("ERROR: Por favor introduzca un nombre que no sea vacío.")
			}

		} else {
			fmt.Println("ERROR: Por favor introduzca un ID que no sea vacío.")
		}

	} else { // Si la opción introducida no es válida
		fmt.Println("ERROR: Por favor introduzca 1 o 2")
		fmt.Println() // Por limpieza
	}

}

/* Función que se conecta al módulo FWQ_Registry para editar o actualizar el perfil de un usuario existente */
func EditarPerfil(ipRegistry, puertoRegistrySockets, puertoRegistryApiRest string) {

	fmt.Println() // Por limpieza
	fmt.Println("**********Modificación de perfil**********")
	fmt.Println() // Por limpieza

	// Damos la posibilidad elegir conexión vía sockets o por API_REST
	fmt.Println("Selecciona el tipo de conexión al registry:")
	fmt.Println("1 -> Sockets")
	fmt.Println("2 -> API_REST")
	fmt.Println() // Por limpieza

	var eleccion int
	fmt.Scan(&eleccion)
	fmt.Println() // Por limpieza

	// Si el usuario elige la conexión por sockets
	if eleccion == 1 {

		cert, err := tls.LoadX509KeyPair("cert/cert.pem", "cert/key.pem")
		if err != nil {
			log.Fatal("Error al cargar los ficheros de certificado y clave asociada: ", err)
		}

		config := tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true, // Para que admita certificados autofirmados
		}

		// conn, err := net.Dial(connType, ipRegistry+":"+puertoRegistry) CONEXIÓN INSEGURA
		conn, err := tls.Dial(connType, ipRegistry+":"+puertoRegistrySockets, &config) // CONEXIÓN SEGURA

		if err != nil {
			fmt.Println("Error al conectarse al Registry:", err.Error())
		} else { // Si el visitante establece conexión con el Registry indicado por parámetro

			defer conn.Close() // Nos aseguramos de cerrar la conexión

			conn.Write([]byte("2" + "\n")) // Le pasamos al Registry la opción elegida por el visitante

			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Información del visitante que se quiere modificar:")
			fmt.Print("Introduce el ID:")
			id, _ := reader.ReadString('\n')

			// Nos aseguramos de que el ID no sea vacío.
			if len(id) > 1 {

				conn.Write([]byte(id))

				fmt.Print("Introduce el nombre:")
				nombre, _ := reader.ReadString('\n')

				// Nos aseguramos de que el nombre no sea vacío.
				if len(nombre) > 1 {

					conn.Write([]byte(nombre))

					fmt.Print("Introduce la contraseña:")
					password, _ := reader.ReadString('\n')

					// Nos aseguramos de que la contraseña no sea vacía.
					if len(password) > 1 {

						conn.Write([]byte(password))

						message, _ := bufio.NewReader(conn).ReadString('\n')

						// Comprobamos si el Registry nos devuelve un mensaje de respuesta
						if message != "" {
							log.Print("Respuesta del Registry: ", message)
						} else {
							log.Print("Lo siento, el Registry no está disponible en estos momentos.")
						}

					} else {
						fmt.Println("ERROR: Por favor introduzca una contraseña que no sea vacía.")
					}

				} else {
					fmt.Println("ERROR: Por favor introduzca un nombre que no sea vacío.")
				}

			} else {
				fmt.Println("ERROR: Por favor introduzca un ID que no sea vacío.")
			}

		}
	} else if eleccion == 2 { // Si el usuario elige la conexión por API_REST

		var id string

		fmt.Print("Introduce tu ID:")
		fmt.Scanln(&id)

		// Nos aseguramos de que no sea válido un id en blanco
		if len(id) > 1 {

			var nombre string

			fmt.Print("Introduce tu nombre:")
			fmt.Scanln(&nombre)

			// Nos aseguramos de que no sea válido un nombre en blanco
			if len(nombre) > 1 {

				var password string

				fmt.Print("Introduce tu contraseña:")
				fmt.Scanln(&password)

				// Nos aseguramos de que no sea válida una contraseña en blanco
				if len(password) > 1 {

					tr := &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Para que admita certificados autofirmados
					}

					// Accedemos al cliente mediante http.Client
					clienteHttp := &http.Client{Transport: tr}

					v := visitante{
						ID:       id,
						Nombre:   nombre,
						Password: password,
					}
					vComoJson, err := json.Marshal(v)
					if err != nil {
						log.Fatalf("Error codificando visitante como JSON: %v", err)
					}

					// Creamos una nueva petición tipo PUT mediante http.NewRequest
					peticion, err := http.NewRequest("PUT", "https://"+ipRegistry+":"+puertoRegistryApiRest+"/editar/"+v.ID, bytes.NewBuffer(vComoJson))
					if err != nil {
						log.Fatalf("Error creando la petición PUT: %v", err)
					}

					// Agregamos las cabeceras que queramos
					peticion.Header.Add("Content-Type", "application/json")

					respuesta, err := clienteHttp.Do(peticion)
					if err != nil {
						fmt.Printf("Error al realizar la petición al servidor API REST: %v\n", err)
						return
					}

					// Nos aseguramos de que se cierra el body recibido
					defer respuesta.Body.Close()

					// Realizamos la lectura del body
					cuerpoRespuesta, err := ioutil.ReadAll(respuesta.Body)
					if err != nil {
						fmt.Printf("Error al leer la respuesta recibida: %v\n", err)
						return
					}

					// Aquí podemos decodificar la respuesta si es un JSON, o convertirla a cadena

					fmt.Println() // Por limpieza
					fmt.Printf("%s", cuerpoRespuesta)
					fmt.Println() // Por limpieza

				} else {
					fmt.Println("ERROR: Por favor introduzca una contraseña que no sea vacía.")
				}

			} else {
				fmt.Println("ERROR: Por favor introduzca un nombre que no sea vacío.")
			}

		} else {
			fmt.Println("ERROR: Por favor introduzca un ID que no sea vacío.")
		}

	} else { // Si la opción introducida no es válida
		fmt.Println("ERROR: Por favor introduzca 1 o 2")
		fmt.Println() // Por limpieza
	}

}

/* Función que envía las credenciales de acceso del visitante para entrar en el parque */
func EntradaParque(IpBroker, PuertoBroker string) {

	fmt.Println("*Bienvenido al parque de atracciones*")

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Por favor introduce tu alias:")
	alias, _ := reader.ReadString('\n')

	if len(alias) > 1 {

		v.ID += strings.TrimSpace(string(alias))

		fmt.Print("y tu password:")
		password, _ := reader.ReadString('\n')

		if len(password) > 1 {

			v.Password += strings.TrimSpace(string(password))

			// SECURIZAMOS LA COMUNICACIÓN EN KAFKA
			// Cargamos la clave de cifrado AES del archivo
			fichero, err := ioutil.ReadFile("claveCifradoAES.txt")
			if err != nil {
				log.Fatal("Error al leer el archivo de la clave de cifrado AES: ", err)
			}

			clave := string(fichero) // Clave de 24 bits

			// Preparamos las credenciales de inicio de sesión del visitante
			mensaje := strings.TrimSpace(string(alias)) + ":" + strings.TrimSpace(string(password)) + ":" + strconv.Itoa(v.Destinox) + "," + strconv.Itoa(v.Destinoy)

			// Ciframos el mensaje
			mensajeCifrado, err := encriptacionAES(mensaje, clave)
			if err != nil {
				panic(err)
			}

			// Mandamos al engine las credenciales de inicio de sesión del visitante para entrar al parque
			productorLogin(IpBroker, PuertoBroker, mensajeCifrado)

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				for sig := range c {
					log.Printf("captured %v, stopping profiler and exiting..", sig)
					mensaje := v.ID + ":" + "OUT" + ":" + strconv.Itoa(v.Destinox) + "," + strconv.Itoa(v.Destinoy)
					mensajeCifrado, err := encriptacionAES(mensaje, clave)
					if err != nil {
						panic(err)
					}
					productorSalir(IpBroker, PuertoBroker, mensajeCifrado)
					fmt.Println()
					fmt.Println("Adios, esperamos que haya disfrutado su estancia en el parque.")
					pprof.StopCPUProfile()
					os.Exit(1)
				}
			}()

			// Recibe del engine el mapa actualizado o un mensaje de parque cerrado
			consumidorLogin(IpBroker, PuertoBroker, clave)

		} else {
			fmt.Println("ERROR: Por favor introduzca un password no vacío.")
		}

	} else {
		fmt.Println("ERROR: Por favor introduzca un ID no vacío.")
	}

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

/* Función que se encarga de enviar las credenciales de inicio de sesión */
func productorLogin(IpBroker, PuertoBroker, credenciales string) {

	var brokerAddress string = IpBroker + ":" + PuertoBroker
	var topic string = "peticiones"

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{brokerAddress},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
		//Dialer:           dialer,
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("Key-Login"),
		Value: []byte(credenciales),
	})
	if err != nil {
		panic("No se pueden encolar las credenciales: " + err.Error())
	}

	fmt.Println("Enviando credenciales -> " + credenciales)

}

/* Función que recibe el mensaje de parque cerrado por parte del engine o no */
func consumidorLogin(IpBroker, PuertoBroker, clave string) {

	respuestaEngine := ""

	broker := IpBroker + ":" + PuertoBroker
	r := kafka.ReaderConfig(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   "respuesta-login",
		GroupID: Ulid(),
		//De esta forma solo cogera los ultimos mensajes despues de unirse al cluster
		StartOffset: kafka.LastOffset,
	})

	reader := kafka.NewReader(r)

	dentroParque := true

	for dentroParque {

		m, err := reader.ReadMessage(context.Background())

		if err != nil {
			panic("Ha ocurrido algún error a la hora de conectarse con kafka: " + err.Error())
		}

		respuestaEngine, err = desencriptacionAES(strings.TrimSpace(string(m.Value)), clave)
		if err != nil {
			panic(err)
		}

		log.Println("Respuesta del engine: " + respuestaEngine)

		if respuestaEngine == (v.ID + ":" + "Acceso concedido") {
			v.DentroParque = 1 // El visitante está dentro del parque
			fmt.Println("El visitante está dentro del parque")
			peticionEntrada := v.ID + ":" + "IN" + ":" + strconv.Itoa(v.Destinox) + "," + strconv.Itoa(v.Destinoy)
			peticionEntradaCifrada, err := encriptacionAES(peticionEntrada, clave)
			if err != nil {
				panic(err)
			}
			productorMovimientos(IpBroker, PuertoBroker, peticionEntradaCifrada) // Le indicamos al engine que el visitante desea entrar al parque
			consumidorMapa(IpBroker, PuertoBroker, clave)
			dentroParque = false
		} else if respuestaEngine == (v.ID + ":" + "Parque cerrado") {
			fmt.Println("Parque cerrado")
			v.DentroParque = 0
			v.ID = ""
			v.Password = ""
			dentroParque = false
		} else if respuestaEngine == (v.ID + ":" + "Aforo al completo") {
			fmt.Println("Aforo al completo")
			v.DentroParque = 0
			v.ID = ""
			v.Password = ""
			dentroParque = false
		}

	}

}

/* Función que actualiza el tiempo de espera de la atracción destino del visitante en base al mapa recibido */
func actualizaAtraccion(mapa [20][20]string) {
	tiempoActualizado, err := strconv.Atoi(mapa[a.Posicionx][a.Posiciony])
	if err != nil { // Si se produce un error al convertir a entero quiere decir que la atracción se ha cerrado
		a.Estado = "Cerrada"
	} else {
		a.TiempoEspera = tiempoActualizado // Sino actualizamos el tiempo sin más
	}
}

/* Función que selecciona una atracción al azar y guarda la posición de dicha atracción en el visitante */
func seleccionaAtraccionAlAzar(mapa [20][20]string) {

	var atraccionesDisponibles []atraccion

	//Elegimos una atracción al azar del mapa entre las que el tiempo de espera sea menor de 60 minutos
	for i := 0; i < 20; i++ {
		for j := 0; j < 20; j++ {
			// Si la posición actual del mapa es un número, con esto nos basta para que no acuda a una atracción cerrada
			if t, err := strconv.Atoi(mapa[i][j]); err == nil {
				if t < 60 { // Si el tiempo de espera es menor a 60 minutos
					atraccionAux := atraccion{
						Posicionx:    i,
						Posiciony:    j,
						TiempoEspera: t,
						Estado:       "Abierta",
					}
					atraccionesDisponibles = append(atraccionesDisponibles, atraccionAux)
				}
			}
		}
	}

	// Elegimos al azar una de las atracciones disponibles
	rand.Seed(time.Now().UnixNano()) // Utilizamos la función Seed(semilla) para inicializar la fuente predeterminada al requerir un comportamiento diferente para cada ejecución
	min := 0
	max := len(atraccionesDisponibles) - 1
	indexAtraccion := (rand.Intn(max-min+1) + min)

	// Actualizamos la coordenadas de destino del visitante
	v.Destinox = atraccionesDisponibles[indexAtraccion].Posicionx
	v.Destinoy = atraccionesDisponibles[indexAtraccion].Posiciony
	a.Posicionx = atraccionesDisponibles[indexAtraccion].Posicionx
	a.Posiciony = atraccionesDisponibles[indexAtraccion].Posiciony
	a.TiempoEspera = atraccionesDisponibles[indexAtraccion].TiempoEspera
	a.Estado = atraccionesDisponibles[indexAtraccion].Estado

	fmt.Println("Me dirijo a la atracción con tiempo de espera igual a " + strconv.Itoa(a.TiempoEspera))

}

/* Función que se encarga de ir moviendo al visitante hasta alcanzar el destino */
func obtenerMovimiento(mapa [20][20]string) string {

	var movimiento string

	// Si el visitante no sabe a qué atracción dirigirse o la atracción actual elegida tiene un tiempo de espera mayor a 60 minutos o está cerrada
	if v.Destinox == -1 || v.Destinoy == -1 || a.TiempoEspera >= 60 || a.Estado == "Cerrada" {
		seleccionaAtraccionAlAzar(mapa)
	} else {
		actualizaAtraccion(mapa) // Actualizamos el tiempo de espera de la atracción destino del visitante
	}

	movimiento = calculaMovimiento() // Obtiene el mejor movimiento en base a las posiciones adyacentes y la atracción destino seleccionada
	actualizaPosicion(movimiento)    // Actualiza la posición actual del visitante en base al mejor movimiento elegido

	// Si el visitante se encuentra en la atracción
	if (v.Posicionx == v.Destinox) && (v.Posiciony == v.Destinoy) {

		time.Sleep(10 * time.Second) // Esperamos un tiempo para simular el tiempo de ciclo de la atracción

		// Ahora el visitante vuelve a desconocer su destino
		v.Destinox = -1
		v.Destinoy = -1
		a.TiempoEspera = -1
		a.Posicionx = -1
		a.Posiciony = -1

	}

	return movimiento

}

/* Función que devuelve el mejor movimiento a realizar en base a la atracción destino elegida por el visitante */
func calculaMovimiento() string {

	var mejorMovimiento string = ""
	var mejorDistancia int
	var nuevaDistancia int

	xOriginal := v.Posicionx
	yOriginal := v.Posiciony

	// Seleccionamos el mejor movimiento para que el visitante alcance su destino
	for i := 0; i < 8; i++ {

		switch i {
		case 0:
			v.Posicionx--
			if v.Posicionx == -1 {
				v.Posicionx = 19
			}
			mejorDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			mejorMovimiento = "N"
			v.Posicionx = xOriginal // Reseteamos la posición
		case 1:
			v.Posicionx++
			if v.Posicionx == 20 {
				v.Posicionx = 0
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "S"
			}
			v.Posicionx = xOriginal // Reseteamos la posición
		case 2:
			v.Posiciony--
			if v.Posiciony == -1 {
				v.Posiciony = 19
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "W"
			}
			v.Posiciony = yOriginal // Reseteamos la posición
		case 3:
			v.Posiciony++
			if v.Posiciony == 20 {
				v.Posiciony = 0
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "E"
			}
			v.Posiciony = yOriginal // Reseteamos la posición
		case 4:
			v.Posicionx--
			v.Posiciony--
			if v.Posicionx == -1 {
				v.Posicionx = 19
			}
			if v.Posiciony == -1 {
				v.Posiciony = 19
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "NW"
			}
			v.Posicionx = xOriginal // Reseteamos la posición
			v.Posiciony = yOriginal // Reseteamos la posición
		case 5:
			v.Posicionx--
			v.Posiciony++
			if v.Posicionx == -1 {
				v.Posicionx = 19
			}
			if v.Posiciony == 20 {
				v.Posiciony = 0
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "NE"
			}
			v.Posicionx = xOriginal // Reseteamos la posición
			v.Posiciony = yOriginal // Reseteamos la posición
		case 6:
			v.Posicionx++
			v.Posiciony--
			if v.Posicionx == 20 {
				v.Posicionx = 0
			}
			if v.Posiciony == -1 {
				v.Posiciony = 19
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "SW"
			}
			v.Posicionx = xOriginal // Reseteamos la posición
			v.Posiciony = yOriginal // Reseteamos la posición
		case 7:
			v.Posicionx++
			v.Posiciony++
			if v.Posicionx == 20 {
				v.Posicionx = 0
			}
			if v.Posiciony == 20 {
				v.Posiciony = 0
			}
			nuevaDistancia = int(math.Abs(float64(v.Destinox)-float64(v.Posicionx))) + int(math.Abs(float64(v.Destinoy)-float64(v.Posiciony))) // Distancia de Manhattan
			if nuevaDistancia < mejorDistancia {
				mejorDistancia = nuevaDistancia
				mejorMovimiento = "SE"
			}
			v.Posicionx = xOriginal // Reseteamos la posición
			v.Posiciony = yOriginal // Reseteamos la posición
		}

	}

	return mejorMovimiento

}

/* Función que actualiza la posición actual del visitante en base al movimiento pasado por parámetro */
func actualizaPosicion(movimiento string) {

	switch movimiento {

	case "N":
		v.Posicionx--
	case "S":
		v.Posicionx++
	case "W":
		v.Posiciony--
	case "E":
		v.Posiciony++
	case "NW":
		v.Posicionx--
		v.Posiciony--
	case "NE":
		v.Posicionx--
		v.Posiciony++
	case "SW":
		v.Posicionx++
		v.Posiciony--
	case "SE":
		v.Posicionx++
		v.Posiciony++
	}

	if v.Posicionx == -1 {
		v.Posicionx = 19
	} else if v.Posicionx == 20 {
		v.Posicionx = 0
	}

	if v.Posiciony == -1 {
		v.Posiciony = 19
	} else if v.Posiciony == 20 {
		v.Posiciony = 0
	}

}

/* Función que se encarga de enviar los movimientos de los visitantes al engine */
func productorMovimientos(IpBroker, PuertoBroker, movimiento string) {

	var brokerAddress string = IpBroker + ":" + PuertoBroker
	var topic string = "peticiones"

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{brokerAddress},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("Key-Moves"),
		Value: []byte(movimiento),
	})
	if err != nil {
		panic("No se puede encolar el movimiento: " + err.Error())
	}

	fmt.Println("Enviando movimiento: " + movimiento)

}

/* Función que se encarga de mandar la solicitud de salida del parque al engine */
func productorSalir(IpBroker, PuertoBroker, peticion string) {

	var brokerAddress string = IpBroker + ":" + PuertoBroker
	var topic string = "peticiones"

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{brokerAddress},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
		//Dialer:           dialer,
	})

	err := w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("Key-Salir"),
		Value: []byte(peticion),
	})
	if err != nil {
		panic("No se puede encolar la solicitud: " + err.Error())
	}

}

/*func (h *HexData) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*h = HexData(decoded)
	return nil
}*/

/* Función que recibe el mapa del engine y lo devuelve formateado */
func consumidorMapa(IpBroker, PuertoBroker, clave string) {

	broker := IpBroker + ":" + PuertoBroker
	r := kafka.ReaderConfig(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   "movimiento-mapa",
		GroupID: Ulid(),
		//De esta forma solo cogera los ultimos mensajes despues de unirse al cluster
		StartOffset: kafka.LastOffset,
	})

	reader := kafka.NewReader(r)

	for v.DentroParque == 1 {

		m, err := reader.ReadMessage(context.Background())

		if err != nil {
			panic("Ha ocurrido algún error a la hora de conectarse con kafka: " + err.Error())
		}

		var mapaObtenido string
		err = json.Unmarshal([]byte(string(m.Value)), &mapaObtenido)

		if err != nil {
			fmt.Printf("Error al decodificar el mapa: %v\n", err)
		}

		// Desciframos el mapa
		mapaDescifrado, err := desencriptacionAES(mapaObtenido, clave)
		if err != nil {
			panic(err)
		}

		// Como el parque ha cerrado tenemos que reiniciar la información del visitante
		if mapaDescifrado == "Engine no disponible" {
			fmt.Println("El engine ha dejado de estar disponible")
			v.DentroParque = 0
			v.ID = ""
			v.Password = ""
			//v.Posicionx = 0
			//v.Posiciony = 0
			//v.Destinox = -1
			//v.Destinoy = -1
		} else {

			// Procesamos el mapa recibido y lo convertimos a un array bidimensional de strings
			//cadenaProcesada := strings.Split(string(m.Value), "|")
			cadenaProcesada := strings.Split(mapaDescifrado, "|")
			var mapa [20][20]string = procesarMapa(cadenaProcesada)
			fmt.Println(mapaDescifrado)
			movimiento := obtenerMovimiento(mapa)
			peticionMovimiento := v.ID + ":" + movimiento + ":" + strconv.Itoa(v.Destinox) + "," + strconv.Itoa(v.Destinoy)
			peticionMovimientoCifrada, err := encriptacionAES(peticionMovimiento, clave)
			if err != nil {
				panic(err)
			}
			productorMovimientos(IpBroker, PuertoBroker, peticionMovimientoCifrada)

			time.Sleep(1 * time.Second) // Mandamos el movimiento del visitante cada segundo

			/*go func() {
				var respuesta string
				fmt.Println("Desea salir del parque (si/no): ")
				fmt.Scanln(&respuesta)
				if respuesta == "s" || respuesta == "S" || respuesta == "si" || respuesta == "SI" || respuesta == "Si" || respuesta == "sI" {
					v.DentroParque = 0
					mensaje := v.ID + ":" + "Salir"
					productorSalir(IpBroker, PuertoBroker, mensaje, ctx)
					fmt.Println()
					fmt.Println("Adios, esperamos que haya disfrutado su estancia en el parque.")
					os.Exit(1)
				}
			}()*/

		}

	}

}

/* Función que formatea el mapa y lo convierte en un array bidimensional de strings */
func procesarMapa(mapa []string) [20][20]string {

	var mapaFormateado [20][20]string

	k := 0

	for i := 0; i < 20; i++ {

		for j := 0; j < 20; j++ {

			if k < 400 {
				mapaFormateado[i][j] = mapa[k]
				k++
			}

		}

	}

	return mapaFormateado

}

/*
* Función que crea el topic para el envio de los movimientos de los visitantes
 */
func crearTopic(IpBroker, PuertoBroker, topic string) {

	var broker1 string = IpBroker + ":" + PuertoBroker
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
			Topic:             topic,
			NumPartitions:     10,
			ReplicationFactor: 1,
		},
	}
	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}

}

func Ulid() string {
	t := time.Now().UTC()
	id := ulid.MustNew(ulid.Timestamp(t), hho.Reader)

	return id.String()
}

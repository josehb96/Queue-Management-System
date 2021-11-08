package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
)

/*
* Estructura del visitante
 */
type visitante struct {
	ID        string `json:"id"`
	Nombre    string `json:"nombre"`
	Password  string `json:"contraseña"`
	Posicionx int    `json:"posicionx"`
	Posiciony int    `json:"posiciony"`
	Destinox  int    `json:"destinox"`
	Destinoy  int    `json:"destinoy"`
	Parque    string `json:"parqueAtracciones"`
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
	Parque       string `json:"parqueAtracciones"`
}

/*
* Estructura del parque
 */
type parque struct {
	ID          string `json:"id"`
	AforoMaximo int    `json:"aforoMaximo"`
	AforoActual int    `json:"aforoActual"`
}

/*
 * @Description : Función main de fwq_engine
 * @Author : Wilmer Fabricio Bravo Shuira
 */
func main() {
	IpKafka := os.Args[1]
	PuertoKafka := os.Args[2]
	numeroVisitantes := os.Args[3]
	IpFWQWating := os.Args[4]
	PuertoWaiting := os.Args[5]

	//Minimo del rand
	min := 0
	//Maximo del rand
	max := 1
	fmt.Println(rand.Intn(max-min) + min)
	fmt.Println("**Bienvenido al engine de la aplicación**")
	fmt.Println("La ip del apache kafka es el siguiente:" + IpKafka)
	fmt.Println("El puerto del apache kafka es el siguiente:" + PuertoKafka)
	fmt.Println("El número máximo de visitantes es el siguiente:" + numeroVisitantes)
	fmt.Println("La ip del servidor de espera es el siguiente:" + IpFWQWating)
	fmt.Println("El puerto del servidor de tiempo es el siguiente:" + PuertoWaiting)

	//Reserva de memoria para el mapa
	var mapa [20][20]string
	//Array de visitantes que se encuentran en el parque
	var visitantesFinales []visitante
	var atraccionesFinales []atraccion
	var parqueTematico []parque
	var conn = conexionBD()
	visitantesFinales, _ = obtenerVisitantesBD(conn)
	atraccionesFinales, _ = obtenerAtraccionesBD(conn)
	parqueTematico, _ = obtenerParqueDB(conn)

	fmt.Println(visitantesFinales)
	fmt.Println(atraccionesFinales)
	fmt.Println(parqueTematico)
	//Ahora obtendremos el visitante y lo mostraremos en el mapa
	//Cada una de las casillas, su valor entero representa el tiempo en minutos de una atracción
	//Cada uno de los personajes tenemos que representarlo por algo
	//Esto se le asignara cuando entre al parque
	//El mapa se carga de la base de datos al arrancar la aplicación
	fmt.Println("*********** FUN WITH QUEUES RESORT ACTIVITY MAP *********")
	fmt.Println("ID   	" + "		Nombre      " + "	Pos.      " + "	Destino")
	//La función Itoa convierte un int a string esto es para que se pueda imprimir por pantalla
	for i := 0; i < len(visitantesFinales); i++ {
		fmt.Println(visitantesFinales[i].ID + "#		" + visitantesFinales[i].Nombre +
			"   #" + "	(" + strconv.Itoa(visitantesFinales[i].Posicionx) + "," + strconv.Itoa(visitantesFinales[i].Posiciony) +
			")" + "   #" + "	(" + strconv.Itoa(visitantesFinales[i].Destinox) + "," + strconv.Itoa(visitantesFinales[i].Destinoy) +
			")")
	}
	mapa = asignacionPosiciones(visitantesFinales, atraccionesFinales, mapa)
	//Matriz transversal bidimensional
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {

			fmt.Print(mapa[i][j], " ")

		}
		fmt.Println()
	}
	conexionKafka()
}

/*
* Función que abre una conexion con la bd
 */
func conexionBD() *sql.DB {
	//Accediendo a la base de datos
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
* Función que obtienen los parques
* @return []parque : Arrays de parque en la base de datos
* @return error : Error en caso de que no se pueda obtener parques
 */
func obtenerParqueDB(db *sql.DB) ([]parque, error) {
	//Cada parque sera un grupo // Idea
	results, err := db.Query("SELECT * FROM parque")
	if err != nil {
		return nil, err //devolvera nil y error en caso de que no se pueda hacer la consulta
	}
	//Cerramos la base de datos
	defer results.Close()
	//Declaramos el array de visitantes
	var parquesTematicos []parque
	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {
		//   var nombreVariable tipoVariable
		//Variable donde guardamos la información de cada una filas de la sentencia
		var parqueTematico parque
		if err := results.Scan(&parqueTematico.ID, &parqueTematico.AforoMaximo,
			&parqueTematico.AforoActual); err != nil {
			return parquesTematicos, err
		}
		//Vamos añadiendo los visitantes al array
		parquesTematicos = append(parquesTematicos, parqueTematico)
	}
	if err = results.Err(); err != nil {
		return parquesTematicos, err
	}
	return parquesTematicos, nil
}

/*
* Función que obtiene todos los visitantes de la bd
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
	var visitantesParque []visitante
	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {
		//   var nombreVariable tipoVariable
		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_visitante visitante
		if err := results.Scan(&fwq_visitante.ID, &fwq_visitante.Nombre,
			&fwq_visitante.Password, &fwq_visitante.Posicionx,
			&fwq_visitante.Posiciony, &fwq_visitante.Destinox, &fwq_visitante.Destinoy,
			&fwq_visitante.Parque); err != nil {
			return visitantesParque, err
		}
		//Vamos añadiendo los visitantes al array
		visitantesParque = append(visitantesParque, fwq_visitante)
	}
	if err = results.Err(); err != nil {
		return visitantesParque, err
	}
	return visitantesParque, nil
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
	//Cerramos la base de datos
	defer results.Close()
	//Declaramos el array de visitantes
	var atraccionesParque []atraccion
	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {
		//   var nombreVariable tipoVariable
		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_atraccion atraccion
		if err := results.Scan(&fwq_atraccion.ID, &fwq_atraccion.TCiclo,
			&fwq_atraccion.NVisitantes, &fwq_atraccion.Posicionx,
			&fwq_atraccion.Posiciony, &fwq_atraccion.TiempoEspera,
			&fwq_atraccion.Parque); err != nil {
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
* Función que asigna los visitantes y los parques en el mapa
* @return [20][20]string : Matriz bidimensional representando el mapa
 */
func asignacionPosiciones(visitantesFinales []visitante, atraccionesFinales []atraccion, mapa [20][20]string) [20][20]string {
	//Asignamos valores a las posiciones del mapa
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(visitantesFinales); k++ {
				if i == visitantesFinales[k].Posicionx && j == visitantesFinales[k].Posiciony {
					mapa[i][j] = "|"
				}
			}
		}
	}
	//Asignamos los valores de tiempo de espera de las atracciones
	//Esto para posicionar una vez esta bien pero los tiempos de espera si
	//que tenemos que actualizarlo
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(atraccionesFinales); k++ {
				if i == atraccionesFinales[k].Posicionx && j == atraccionesFinales[k].Posiciony {
					mapa[i][j] = strconv.Itoa(atraccionesFinales[k].TiempoEspera)
				}
			}
		}
	}
	return mapa
}

/**
* Función que se conecta al servidor de tiempo de espera
**/
func tiempoEspera(IpFWQWating, PuertoWaiting string) {
	fmt.Println("***Tiempo de espera***")
	var connType string = "tcp"
	conn, err := net.Dial(connType, IpFWQWating+":"+PuertoWaiting)
	if err != nil {
		fmt.Println("Error al conectarse:", err.Error())
		os.Exit(1)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("***Actualizando los tiempos de espera***")
		//Leer entrada hasta nueva linea, introduciendo llave
		input, _ := reader.ReadString('\n')
		//Enviamos la conexion del socket
		conn.Write([]byte(input))
		//Escuchando por el relay
		message, _ := bufio.NewReader(conn).ReadString('\n')
		//Print server relay
		log.Print("Server relay:", message)
	}
}

/**
* Función que conecta el engine con el kafka
**/
func conexionKafka() {
	//Configuración de lector de kafka
	conf := kafka.ReaderConfig{
		//El broker habra que cambiarlo por otro
		Brokers:  []string{"localhost:9092"},
		Topic:    "sd-events", //Topico que hemos creado
		GroupID:  "g1",
		MaxBytes: 10,
	}
	reader := kafka.NewReader(conf)
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Ha ocurrido algún error a la hora de conectarse con kafka", err)
			continue
		}
		fmt.Println("El mensaje es desde el terminal wilmer : ", string(m.Value))
	}
}

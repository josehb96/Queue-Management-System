package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

	//Creamos el topic...Cambiar la Ipkafka en la función principal
	//Si no se ejecuta el programa, se cierra el kafka?
	crearTopics(IpKafka, PuertoKafka, "mapa-visitantes")
	crearTopics(IpKafka, PuertoKafka, "movimientos-visitantes")

	fmt.Println("**Bienvenido al engine de la aplicación**")
	fmt.Println("La ip del apache kafka es el siguiente:" + IpKafka)
	fmt.Println("El puerto del apache kafka es el siguiente:" + PuertoKafka)
	fmt.Println("El número máximo de visitantes es el siguiente:" + numeroVisitantes)
	fmt.Println("La ip del servidor de espera es el siguiente:" + IpFWQWating)
	fmt.Println("El puerto del servidor de tiempo es el siguiente:" + PuertoWaiting)
	fmt.Println()

	//Reserva de memoria para el mapa
	var mapa [20][20]string

	// Visitantes, atracciones que se encuentran en el parque
	var visitantes []visitante
	var atracciones []atraccion
	var parqueTematico []parque

	var conn = conexionBD()
	maxVisitantes, _ := strconv.Atoi(numeroVisitantes)
	establecerMaxVisitantes(conn, maxVisitantes)

	for {

		visitantes, _ = obtenerVisitantesBD(conn)
		atracciones, _ = obtenerAtraccionesBD(conn)
		parqueTematico, _ = obtenerParqueDB(conn)

		// Esta parte la podemos suprimir, simplemente es a modo de comprobación
		fmt.Println("Visitantes en el parque: ")
		fmt.Println(visitantes)
		fmt.Println() // Para mejorar la salida por pantalla
		fmt.Println("Atracciones del parque: ")
		fmt.Println(atracciones)
		fmt.Println() // Para mejorar la salida por pantalla
		fmt.Println("Información del parque: ")
		fmt.Println(parqueTematico)
		fmt.Println() // Para mejorar la salida por pantalla

		//QUERY DELETE.....DELETE FROM visitante WHERE id = h7;
		//Ahora obtendremos el visitante y lo mostraremos en el mapa
		//Cada una de las casillas, s valor entero representa el tiempo en minutos de una atracción
		//Cada uno de los personajes tenemos que representarlo por algo
		//Esto se le asignara cuando entre al parque
		//El mapa se carga de la base de datos al arrancar la aplicación

		fmt.Println("*********** FUN WITH QUEUES RESORT ACTIVITY MAP *********")
		fmt.Println("ID   	" + "		Nombre      " + "	Pos.      " + "	Destino")

		// Hay que usar la función TrimSpace porque al parecer tras la obtención de valores de BD se agrega un retorno de carro a cada variable
		for i := 0; i < len(visitantes); i++ {
			fmt.Println(strings.TrimSpace(visitantes[i].ID) + " 		" + strings.TrimSpace(visitantes[i].Nombre) +
				"    " + "	(" + strings.TrimSpace(strconv.Itoa(visitantes[i].Posicionx)) + "," + strings.TrimSpace(strconv.Itoa(visitantes[i].Posiciony)) +
				")" + "    " + "	(" + strings.TrimSpace(strconv.Itoa(visitantes[i].Destinox)) + "," + strings.TrimSpace(strconv.Itoa(visitantes[i].Destinoy)) +
				")")
		}

		//Obtenidos todos los visitantes y las atracciones del parque en la BD, asignamos cada uno a la posición que debe tener
		mapa = asignacionPosiciones(visitantes, atracciones, mapa)

		//Para empezar con el kafka
		//ctx := context.Background()
		//var mapa1D []byte = convertirMapa(mapa)
		//consumidorEngineKafka(IpKafka, PuertoKafka)
		//productorEngineKafkaVisitantes(IpKafka, PuertoKafka, ctx, mapa1D)

		// Cada X segundos se conectará al servidor de tiempos para actualizar los tiempos de espera de las atracciones
		time.Sleep(time.Duration(5 * time.Second))
		tiempo := "Mándame los tiempos de espera actualizados"
		conexionTiempoEspera(conn, IpFWQWating, PuertoWaiting, tiempo)

		//Aqui podemos hacer un for y que solo se envien la información de un visitante por parametro
		//Matriz transversal bidimensional

		for i := 0; i < len(mapa); i++ {
			for j := 0; j < len(mapa[i]); j++ {

				fmt.Print(" " + mapa[i][j])

			}
			fmt.Println()
		}

	}

}

/*
* Función que convierte el mapa de tipo string en []byte
* @return []mapa : Retorna un array de bytes que van a hacer enviados a los visitantes
 */
func convertirMapa(mapa [20][20]string) []byte {
	var mapaOneD []byte
	var cadenaMapa []byte
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa); j++ {
			cadenaMapa = []byte(mapa[i][j])
			mapaOneD = append(mapaOneD, cadenaMapa...)
		}
	}
	return mapaOneD
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
* Función que obtiene aquellos visitantes de la BD que se encuentren en el parque
* @return []visitante : Arrays de los visitantes obtenidos en la sentencia
* @return error : Error en caso de que no se haya podido obtener ninguno
 */
func obtenerVisitantesBD(db *sql.DB) ([]visitante, error) {

	//Ejecutamos la sentencia
	results, err := db.Query("SELECT * FROM visitante WHERE dentroParque = 1")

	if err != nil {
		return nil, err //devolvera nil y error en caso de que no se pueda hacer la consulta
	}

	//Cerramos la base de datos
	defer results.Close()

	//Declaramos el array de visitantes
	var visitantesParque []visitante

	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {

		//Variable donde guardamos la información de cada una filas de la sentencia
		var fwq_visitante visitante

		if err := results.Scan(&fwq_visitante.ID, &fwq_visitante.Nombre,
			&fwq_visitante.Password, &fwq_visitante.Posicionx,
			&fwq_visitante.Posiciony, &fwq_visitante.Destinox, &fwq_visitante.Destinoy,
			&fwq_visitante.DentroParque, &fwq_visitante.Parque); err != nil {
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

	//Declaramos el array de atracciones
	var atraccionesParque []atraccion

	//Recorremos los resultados obtenidos por la consulta
	for results.Next() {

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
func asignacionPosiciones(visitantes []visitante, atracciones []atraccion, mapa [20][20]string) [20][20]string {

	//Asignamos valores a las posiciones del mapa
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(visitantes); k++ {
				if i == visitantes[k].Posicionx && j == visitantes[k].Posiciony {
					mapa[i][j] = "X"
				}
			}
		}
	}

	//Asignamos los valores de tiempo de espera de las atracciones
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			for k := 0; k < len(atracciones); k++ {
				if i == atracciones[k].Posicionx && j == atracciones[k].Posiciony {
					mapa[i][j] = strconv.Itoa(atracciones[k].TiempoEspera)
				}
			}
		}
	}

	// Las casillas del mapa que no tengan ni visitantes ni atracciones las representamos con una guión
	for i := 0; i < len(mapa); i++ {
		for j := 0; j < len(mapa[i]); j++ {
			if len(mapa[i][j]) == 0 {
				mapa[i][j] = "-"
			}
		}
	}

	return mapa

}

/*
* Función que se conecta al servidor de tiempos para obtener los tiempos de espera actualizados
 */
func conexionTiempoEspera(db *sql.DB, IpFWQWating, PuertoWaiting, tiempo string) {

	fmt.Println("***Conexión con el servidor de tiempo de espera***")
	fmt.Println("Arrancando el engine para atender los tiempos en el puerto: " + IpFWQWating + ":" + PuertoWaiting)
	var connType string = "tcp"
	conn, err := net.Dial(connType, IpFWQWating+":"+PuertoWaiting)

	if err != nil {
		fmt.Println("Error a la hora de escuchar", err.Error())
	} else {
		fmt.Println("***Actualizando los tiempos de espera***")

		conn.Write([]byte("Mándame los tiempos de espera actualizados" + "\n")) // Mandamos la petición
		tiemposEspera, _ := bufio.NewReader(conn).ReadString('\n')              // Obtenemos los tiempos de espera actualizados

		arrayTiemposEspera := strings.Split(tiemposEspera[:len(tiemposEspera)-1], "|")

		// Actualizamos los tiempos de espera de las atracciones en la BD
		actualizaTiemposEsperaBD(db, arrayTiemposEspera)

		if tiemposEspera != "" {
			log.Println("Tiempos de espera actualizados: " + tiemposEspera)
		} else {
			log.Println("Servidor de tiempos no disponible.")
		}

	}

}

/* Función que establece en la tabla parque de la BD el aforo máximo permitido */
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

/**
* Función que conecta el engine con el kafka
**/
func consumidorEngineKafka(IpKafka, PuertoKafka string) {

	puertoKafka := IpKafka + ":" + PuertoKafka

	//Configuración de lector de kafka
	conf := kafka.ReaderConfig{
		//El broker habra que cambiarlo por otro
		Brokers:  []string{puertoKafka},
		Topic:    "visitantes-engine", //Topico que hemos creado
		MaxBytes: 10,
	}

	reader := kafka.NewReader(conf)

	sigue := true
	for sigue {

		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Ha ocurrido algún error a la hora de conectarse con kafka", err)
			continue
		}

		fmt.Println("Mensaje desde el gestor de colas: ", string(m.Value))
		sigue = false
	}

}

/*
* Función que envia el mapa a los visitantes
 */
func productorEngineKafkaVisitantes(IpBroker, PuertoBroker string, ctx context.Context, mapa []byte) {
	var broker1Addres string = IpBroker + ":" + PuertoBroker
	var broker2Addres string = IpBroker + ":" + PuertoBroker
	var topic string = "mapa-visitantes"
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{broker1Addres, broker2Addres},
		Topic:            topic,
		CompressionCodec: kafka.Snappy.Codec(),
	})
	//Para finalizar la iteración del bucle
	sigue := true
	for sigue {

		//https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol
		//https://docs.confluent.io/clients-confluent-kafka-go/current/overview.html
		//https://www.confluent.io/blog/5-things-every-kafka-developer-should-know/

		//https://en.m.wikipedia.org/wiki/SerDes
		//Compresion de mensajes
		//https://developer.ibm.com/articles/benefits-compression-kafka-messaging/
		err := w.WriteMessages(ctx, kafka.Message{
			Key:   []byte("Key-A"), //[]byte(strconv.Itoa(i)),
			Value: []byte(mapa),
		})
		if err != nil {
			panic("No se puede escribir mensaje" + err.Error())
		}
		//Descanso
		time.Sleep(time.Second)
		sigue = false
	}
}

/* Función que actualiza los tiempos de espera de las atracciones en la BD*/
func actualizaTiemposEsperaBD(db *sql.DB, tiemposEspera []string) {

	results, err := db.Query("SELECT * FROM atraccion")

	// Comrpobamos que no se produzcan errores al hacer la consulta
	if err != nil {
		panic("Error al hacer la consulta a la BD: " + err.Error())
	}

	defer results.Close() // Nos aseguramos de cerrar

	i := 0

	// Recorremos todas las filas de la consulta
	for results.Next() {

		// MODIFICAMOS la información de dicho visitante en la BD
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

		fmt.Println("Atracción modificada.")

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
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}
	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}
}

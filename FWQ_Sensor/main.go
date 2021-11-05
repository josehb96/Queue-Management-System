package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

type sensor struct {
	IdAtraccion int
	Personas    int
}

func main() {

	ipBrokerGestorColas := os.Args[1]

	puertoBrokerGestorColas := os.Args[2]

	idAtraccion, err := strconv.Atoi(os.Args[3])

	if err != nil {
		panic("Error: Introduzca por parámetros IP, PUERTO e ID " + err.Error())
	}

	brokerAddress := ipBrokerGestorColas + ":" + puertoBrokerGestorColas

	// Creamos un sensor
	s := new(sensor)
	s.IdAtraccion = idAtraccion
	// Generamos un número aleatorio de personas
	rand.Seed(time.Now().UnixNano()) // Utilizamos la función Seed(semilla) para inicializar la fuente predeterminada al requerir un comportamiento diferente para cada ejecución
	min := 0
	max := 10
	s.Personas = (rand.Intn(max-min+1) + min)
	fmt.Printf("Sensor creado para la atracción %d en la que inicialmente hay %d personas\n", s.IdAtraccion, s.Personas)

	// Generamos un tiempo aleatorio entre 1 y 3 segundos
	rand.Seed(time.Now().UnixNano()) // Utilizamos la función Seed(semilla) para inicializar la fuente predeterminada al requerir un comportamiento diferente para cada ejecución
	min = 1
	max = 3
	tiempoAleatorio := (rand.Intn(max-min+1) + min)

	// Envíamos al servidor de tiempos el número de personas que se encuentra en la cola de la atracción
	enviaInformacion(s, brokerAddress, tiempoAleatorio)

}

/* Función que envía mediante un productor de Kafka la información recogida por el sensor  */
func enviaInformacion(s *sensor, brokerAddress string, tiempoAleatorio int) {

	// Inicializamos el escritor
	escritor := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9094"},
		Topic:   "sensor-tiempos",
	})

	for {

		err := escritor.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte("Atraccion " + strconv.Itoa(s.IdAtraccion)),
				Value: []byte(strconv.Itoa(s.IdAtraccion) + ":" + strconv.Itoa(s.Personas)),
			})

		if err != nil {
			panic("Error: No se puede escribir el mensaje: " + err.Error())
		}

		// Generamos un número aleatorio de personas
		rand.Seed(time.Now().UnixNano()) // Utilizamos la función Seed(semilla) para inicializar la fuente predeterminada al requerir un comportamiento diferente para cada ejecución
		min := 0
		max := 10
		s.Personas = (rand.Intn(max-min+1) + min)

		// Cada x segundos el sensor envía la información al servidor de tiempos
		time.Sleep(time.Duration(tiempoAleatorio) * time.Second)
	}

}

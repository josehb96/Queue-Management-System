package main

import (
	"fmt"
	"strconv"
	"strings"
)

func sumar(expresion string) int {

	// 4+4+5+8
	valores := strings.Split(expresion, "+")

	var result int

	for _, valor := range valores {
		num, error := strconv.Atoi(valor)

		if error != nil {
			fmt.Println(error)
			fmt.Println("Error: Tiene que ingresar un número entero")
			fmt.Println("o !Solo debes realizar con operador +!!")
		} else {
			result += num
		}

	}

	return result

}

func main() {

	var expresion string
	var result int

	fmt.Printf("=>")
	fmt.Scanln(&expresion)

	result = sumar(expresion)

	fmt.Printf("=> %d\n", result)

}

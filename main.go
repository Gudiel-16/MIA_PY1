package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func main() {
	leerEntrada()
}

//cuando analice texto de entrada se iran guardando aca los comandos
var listaComandos []string

//--------------------------------ESTRUCTURAS-------------------------------//

//mbr que tendra cada archivo creado
type mbr struct {
	Tamanio       int64
	Fecha         [20]byte
	NumAsignacion int64
	Particiones   [4]NodoParticion
}

//NodoParticion ,
type NodoParticion struct {
	Estado             byte
	TipoParticion      byte
	TipoAjuste         [2]byte
	Tamanio            int64
	Name               [16]byte
	ParticionesLogicas [5]NodoParticionLogica //sera funcional solo para las extendidas
	Start              int64                  //byte donde inicia la particion
}

//NodoParticionLogica ,
type NodoParticionLogica struct {
	Estado        byte
	TipoParticion byte
	TipoAjuste    [2]byte
	Tamanio       int64
	Name          [16]byte
	Start         int64 //byte donde inicia la particion
	End           int64 //byte donde termina la particion
}

//--------------------------------FINAL ESTRUCTURAS-------------------------------//

//leera los comandos de entrada (los que escribe el usuario)
func leerEntrada() {

	var enviar bool = false
	var concatenar string = ""
	for true {
		fmt.Print("\n[ nuevo comando ]% ")
		lectura := bufio.NewReader(os.Stdin)
		entrada, _ := lectura.ReadString('\n')         // Leer hasta el separador de salto de línea
		eleccion := strings.TrimRight(entrada, "\r\n") // Remover el salto de línea de la entrada del usuario

		if strings.HasSuffix(eleccion, "\\*") { //si al final termina con \* (seguira concatenando)
			concatenar = concatenar + eleccion
			enviar = false
		} else { //cuando la ultima linea no tengo al final \* entonces ejecutara
			concatenar = concatenar + eleccion
			enviar = true
		}

		if eleccion == "exit" { //para salir de ejecucion
			break
		}

		if enviar == true { //para empezar analizar
			analizador(concatenar)
			//imprimirListaComandos() //************************************************************
			logica()
			vaciarListaComandos()
			concatenar = ""
			enviar = false
		}
	}
}

func analizador(cadena string) {
	var estado int = 0
	var examinarAsci int = 0
	var caracter string = ""

	for i := 0; i < len(cadena); i++ {
		examinar := cadena[i]        //caracter actual de la cadena
		examinarAsci = int(examinar) //numero asci del caracter actual

		switch estado {
		case 0:
			if examinarAsci == 10 { //salto de linea
				estado = 0
			} else if examinarAsci == 32 { //espacio en blanco
				estado = 0
			} else if examinarAsci == 9 { //tabulacion
				estado = 0
			} else if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 34 { // ""
				caracter = caracter + string(examinar)
				estado = 4 //entra en estado de rutas
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 58 { // :
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 45 { // -
				caracter = caracter + string(examinar)
				estado = 2
			} else if examinarAsci == 62 { // >
				caracter = caracter + string(examinar)
				estado = 2
			} else if examinarAsci == 35 { // #
				estado = 3 //pasa a estado de comentario
			}
			break
		case 1:
			if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 34 { // "
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 58 { // :
				caracter = caracter + string(examinar)
				estado = 1
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break
		case 2:
			if examinarAsci == 45 { // -
				caracter = caracter + string(examinar)
				estado = 2
			} else if examinarAsci == 62 { // >
				caracter = caracter + string(examinar)
				estado = 2
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break
		case 3:
			if examinarAsci == 10 { //salto de linea
				//i = i - 1
				estado = 0
			} else {
				estado = 3
			}

			break
		case 4:
			if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 34 { // "
				caracter = caracter + string(examinar) //concateno
				estado = 4
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 32 { //espacio en blanco
				caracter = caracter + " "
				estado = 4
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break

		}
	}

	if caracter != "" {
		analizador2(caracter) //para que envie la ultima palabra
	}

}

//ira guardando comandos en la lista
func analizador2(cadena string) {
	listaComandos = append(listaComandos, cadena)
}

func imprimirListaComandos() {
	for i := 0; i < len(listaComandos); i++ {
		fmt.Println(listaComandos[i])
	}
}

func vaciarListaComandos() {
	listaComandos = nil
}

func logica() {
	for i := 0; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "exec") == 0 {
			execComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "mkdisk") == 0 {
			mkdiskComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "rmdisk") == 0 {
			rmdiskComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "fdisk") == 0 {
			fdiskComando(i)
		}
	}
}

//--------------------------------INICIO EXEC-------------------------------//
//recibe el parametro index, que es el indice por donde actualmente se esta
func execComando(index int) {

	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					leerArchivoExec(ruta2)         //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					leerArchivoExec(ruta)
				}
			} else {
				fmt.Print("\n[ ERROR: mal formato del comando 'exec' ]")
			}
		}
	}
}

//lee el texto que contiene un archivo
func leerArchivoExec(ruta string) {
	texto, err := ioutil.ReadFile(ruta) // just pass the file name, ej. /home/gudiel/z.txt
	if err != nil {
		fmt.Print(err)
	}

	//fmt.Println(b)   // imprimir contenido en'bytes'
	str := string(texto) // convertir a 'string'
	//fmt.Println(str)
	lecturaLineaLineaDeArchivo(str)
}

//ira ejecutando los comandos de las lineas leidas
func lecturaLineaLineaDeArchivo(texto string) {
	var estado int = 0
	var examinarAsci int = 0
	var caracteres string = "" //ira concantenando carecteres de linea actual
	//var comandos string = ""   // sera la linea o lineas de comandos

	for i := 0; i < len(texto); i++ {
		examinar := texto[i]         //caracter actual de la cadena
		examinarAsci = int(examinar) //numero asci del caracter actual

		switch estado {
		case 0:
			if examinarAsci == 10 { //salto de linea, quiere decir que finalizo una linea de comandos
				caracteres = caracteres + string(examinar)

				if strings.Contains(strings.ToLower(caracteres), "pause") {
					//fmt.Println(caracteres)
					vaciarListaComandos() //al principio porque el usuario al inicio ejecuta un exec, entonces se vacia para que no entre en ciclo infinito
					analizador(caracteres)
					//imprimirListaComandos()
					logica()
					vaciarListaComandos()
					caracteres = ""
					fmt.Print("\n[ pause: presione 'enter' ]")
					bufio.NewReader(os.Stdin).ReadBytes('\n')
				}

			} else { //ira concatenando cualquier cosa que no sea salto de linea
				caracteres = caracteres + string(examinar)
			}
		}
	}

	//para que al final envie los ultimos comandos, (ya que de ultimo no abra un proximo pause)
	vaciarListaComandos() //al principio porque el usuario al inicio ejecuta un exec, entonces se vacia para que no entre en ciclo infinito
	analizador(caracteres)
	//imprimirListaComandos()
	logica()
	vaciarListaComandos()
	caracteres = ""
}

//--------------------------------FIN EXEC----------------------------//

//-------------------------------INICIO MKDISK-------------------------------//
//MKDISK (crea disco duro)
func mkdiskComando(index int) {

	var size int = 0
	path := ""
	name := ""
	var unit string = "m"

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "size") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -size->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				size = tam
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en size de MKDISK ]")
				}

			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'Size' ]")
			}

		} else if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'name' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "unit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				unit = listaComandos[i+2] //toma el string
			} else {
				fmt.Print("\n[ ERROR:  Se ha producido un error con el comando 'MKDISK' -> 'unit' ]")
			}
		}
	}

	crearArchivo(size, path, name, unit)

}

//crea el archivo (disco)
func crearArchivo(size int, path string, name string, unit string) {

	//hacemos las operaciones para definir el tamanio
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		size = size * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		size = size * 1024 * 1024
	}

	ruta := path + name //concatenamos la ruta y el nombre del archivo
	file, err := os.Create(ruta)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	//creamos una variable que sera un 0
	var otro int8 = 0
	s := &otro

	// --------------------------------DEFINIMOS EL TAMANIO DE NUESTRO ARCHIVO-------------------------------- //
	//Escribimos un 0 en el inicio del archivo
	var binario bytes.Buffer
	binary.Write(&binario, binary.BigEndian, s)
	escribirBytes(file, binario.Bytes())

	//Nos posicionamos en la ultima posicion
	file.Seek(int64(size-1), 0) // segundo parametro: 0, 1, 2.     0 -> Inicio, 1-> desde donde esta el puntero, 2 -> Del fin para atras
	//Escribimos un 0 al final del archivo.
	var binario2 bytes.Buffer
	binary.Write(&binario2, binary.BigEndian, s)
	escribirBytes(file, binario2.Bytes())

	//-------------------------------------------------------------------------------------------------------------//

	//Escribimos nuestro struct (MBR) en el inicio del archivo
	file.Seek(0, 0) // nos posicionamos en el inicio del archivo.

	//Asignamos valores a los atributos del struct.
	disco := mbr{}

	//tamanio disco
	disco.Tamanio = int64(size)

	t := time.Now()
	fecha := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	// Igualar cadenas a array de bytes (array de chars)
	copy(disco.Fecha[:], fecha)

	//numero de asignacion aleatorio
	disco.NumAsignacion = int64(rand.Intn(100))

	s1 := &disco

	//Escribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//Método para escribir en un archivo
func escribirBytes(file *os.File, bytes []byte) {
	_, err := file.Write(bytes)

	if err != nil {
		log.Fatal(err)
	}
}

//-------------------------------FIN MKDISK-------------------------------//

//-------------------------------INICIO RMDISK-------------------------------//
//RMDISK (elimina un archivo)
func rmdiskComando(index int) {
	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					os.Remove(ruta2)
				} else { //sino tiene comillas manda la ruta normal
					os.Remove(ruta)
				}
			} else {
				fmt.Print("\n[ ERROR: formato del comando 'rmdisk' ]")
			}
		}
	}
}

//-------------------------------FIN RMDISK-------------------------------//

//-------------------------------INICIO FDISK-------------------------------//
//FDISK (administra particiones del archivo, ajustes)
func fdiskComando(index int) {

	var size int64 = 0
	unit := "k"
	path := ""
	typee := "p"
	fit := "wf"
	deletee := ""
	name := ""
	add := 0

	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "size") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -size->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				size = int64(tam)
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en 'size' de 'FDISK' ]")
				}

			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'size' ]")
			}

		} else if strings.Compare(strings.ToLower(listaComandos[i]), "unit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				unit = listaComandos[i+2] //toma el string
			} else {
				fmt.Print("\n[ ERROR:  Se ha producido un error con el comando 'FDISK' -> 'unit' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "type") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				typee = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'type' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "fit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				fit = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'fit' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "delete") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				deletee = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'delete' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'name' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "add") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				add = tam
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en 'size' de 'FDISK' ]")
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'add' ]")
			}
		}
	}
	operacionFdisk(size, unit, path, typee, fit, deletee, name, add)
}

func operacionFdisk(size int64, unit string, path string, typee string, fit string, deletee string, name string, add int) {
	if strings.Compare(deletee, "") != 0 { //si hay que eliminar una particion

	} else if add != 0 { //agregar o quitar espacio de particion

	} else { //crea una particion
		agregarParticion(size, unit, path, typee, fit, name)
	}
}

func agregarParticion(size int64, unit string, path string, typee string, fit string, name string) {

	if validarLimiteDeParticionesEnDisco(path) { //si se puede agregar otra particion
		if strings.Compare(strings.ToLower(typee), "p") == 0 { //si es primaria
			if validarQueTengaEspacioElDisco(path, size, unit) { //si el disco aun tiene espacio
				insertarParticionPrimaria(path, size, typee, fit, name, unit)
			} else {
				fmt.Print("\n[ ERROR: no hay espacio para agregar la particion primaria: ", name, " ]")
			}
		} else if strings.Compare(strings.ToLower(typee), "e") == 0 { //si es extendida

		} else if strings.Compare(strings.ToLower(typee), "l") == 0 { //si es logica

		}
	} else {
		fmt.Print("\n[ ERROR: Ya alcanzo el limite de de particiones en el disco: ]")
	}
}

//valida ya que el disco puede tener maximo 4 particiones
func validarLimiteDeParticionesEnDisco(path string) bool {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	contador := 0

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio == 0 {
			contador++
		}
	}

	//si hay algun espacio para otra particion
	if contador > 0 {
		return true //retorna que hay espacio
	}

	return false
}

//valida si hay un espacio en el mbr para particion primaria
func validarQueTengaEspacioElDisco(path string, sizeParticion int64, unit string) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizeParticion = sizeParticion * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizeParticion = sizeParticion * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	//sera la sumatoria de byte de todas las particiones
	var contadorSize int64 = 0

	//recorro para sumar todos los byte de las particiones
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio != 0 {
			contadorSize = contadorSize + int64(actual.Tamanio)
		}
	}

	//espacio disponible = (tamanio disco) - (espacio actual de todas las particiones)
	var espacioDisponible int64 = int64(m.Tamanio) - contadorSize

	//si hay espacio aun
	if sizeParticion <= espacioDisponible {
		return true //retorna que hay espacio
	}

	return false
}

func insertarParticionPrimaria(path string, sizePart int64, typee string, fit string, name string, unit string) {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizePart = sizePart * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizePart = sizePart * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//accedo a las particiones
	misParticiones := m.Particiones
	fmt.Println("arr pos 0 : ", misParticiones[0].Tamanio)
	fmt.Println("arr pos 1 : ", misParticiones[1].Tamanio)
	fmt.Println("arr pos 2 : ", misParticiones[2].Tamanio)
	fmt.Println("arr pos 3 : ", misParticiones[3].Tamanio)

	contador := 0

	//recorro para ver cuual esta vacia
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio == 0 {
			contador = i
			break
		}
	}

	//se inserta despues del MBR
	if contador == 0 {
		fmt.Println("PRIMARA POSICION")
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 's'
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)
		particionPrimariaNew.Start = int64(size) + 1

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
		fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)

		//pueden ser en la posicion 1, 2, 3
	} else if contador > 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 's'
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)

		//Donde empieza? empieza donde termina la particion anterior
		inicioPartAnt := misParticiones[contador-1].Start //byte donde inicia la particion anterior
		tamPartAnt := misParticiones[contador-1].Tamanio  //tamanio de la particion anterior
		finPartAnt := inicioPartAnt + tamPartAnt          //byte donde finaliza la particion anterior
		particionPrimariaNew.Start = finPartAnt + 1       //aqui empieza la nueva particion

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
		fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)
	}

	fmt.Println("arr pos 0 : ", misParticiones[0].Tamanio)
	fmt.Println("arr pos 1 : ", misParticiones[1].Tamanio)
	fmt.Println("arr pos 2 : ", misParticiones[2].Tamanio)
	fmt.Println("arr pos 3 : ", misParticiones[3].Tamanio)

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se creo una nueva particion se agregadron a 'misPartiiones'
	//entonces 'misParticiones' tiene las actuales, mas la nueva que se le inserto
	//por eso se iguala de nuevo, para que em 'm.Particiones' se guarden particiones ya actualizadas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//Función que lee del archivo, se especifica cuantos bytes se quieren leer.
func leerBytesFdisk(file *os.File, number int) []byte {
	bytes := make([]byte, number) //array de bytes, de tamanio que recibe

	_, err := file.Read(bytes) // Leido -> bytes
	if err != nil {
		log.Fatal(err)
	}

	return bytes
}

//-------------------------------FIN FDISK-------------------------------//

//-------------------------------INICIO MOUNT-------------------------------//
func mountComando(index int) {
	path := ""
	name := ""

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'name' ]")
			}
		}
	}

	montarParticion(path, name)
}

func montarParticion(path string, name string) {

}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------INICIO UNMOUNT-------------------------------//
func unmountComando(index int) {
	idn := ""

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "idn") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				idn = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'name' ]")
			}
		}
	}

	desmontarParticion(idn)
}

func desmontarParticion(path string) {

}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------OPERACIONES PARA MBR-------------------------------//

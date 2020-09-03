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
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	//leerEntrada()
	//reporteMBR("/home/gudiel/Disco1.dsk")
	reporteDisk("/home/gudiel/Disco1.dsk")
	//pruebaMount()
	//m := mbr{}
	//fmt.Println(int(unsafe.Sizeof(m)))
	//fmt.Println(int(binary.Size(m)))

}

//cuando analice texto de entrada se iran guardando aca los comandos
var listaComandos []string

//se iran guardando los mount
var mapaMount = make(map[string][]NodoMount)

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
	Estado             uint8
	TipoParticion      byte
	TipoAjuste         [2]byte
	Tamanio            int64
	Name               [16]byte
	ParticionesLogicas [15]NodoParticionLogica //sera funcional solo para las extendidas
	Start              int64                   //byte donde inicia la particion
}

//NodoParticionLogica ,
type NodoParticionLogica struct {
	Estado        uint8
	TipoParticion byte
	TipoAjuste    [2]byte
	Tamanio       int64
	Name          [16]byte
	Start         int64 //byte donde inicia la particion
	Next          int64 //byte donde termina la particion
}

//NodoMount ,
type NodoMount struct {
	Path   string
	Name   string
	PartID string
}

//SuperBoot ,
type SuperBoot struct {
	SbNombre                                     [16]byte
	SbCantidadEstructurasArbolDirectorio         int64
	SbCantidadEstructurasDetalleDirectorio       int64
	SbCantidadInodos                             int64
	SbCantidadBloques                            int64
	SbCantidadEstructurasArbolDirectorioLibres   int64
	SbCantidadEstructurasDetalleDirectorioLibres int64
	SbCantidadInodosLibres                       int64
	SbCantidadBloquesLibres                      int64
	SbFechaCreacion                              [20]byte
	SbFechaUltimoMontaje                         [20]byte
	SbContadorMontajes                           int64
	SbApInicioBitmapArbolDirectorio              int64
	SbApInicioArbolDirectorio                    int64
	SbApInicioBitmapDetalleDirectorio            int64
	SbApInicioDetalleDirectorio                  int64
	SbApInicioBitmapInodos                       int64
	SbApInicioInodos                             int64
	SbApInicioBitmapBloques                      int64
	SbApInicioBloques                            int64
	SbApInicioBitacora                           int64
	SbTamanioEstructuraArbolDirectorio           int64
	SbTamanioEstructuraDetalleDirectorio         int64
	SbTamanioEstructuraInodo                     int64
	SbTamanioEstructuraBloques                   int64
	SbPrimerBitLibreArbolDirectorio              int64
	SbPrimerBitLibreDetalleDirectorio            int64
	SbPrimerBitLibreInodos                       int64
	SbPrimerBitLibreBloques                      int64
	SbMagigNum                                   int64
}

// ArbolVirtualDirectorio ,
type ArbolVirtualDirectorio struct {
	AvdNombreDirectorio    [16]byte
	AvdFechaCreacion       [20]byte
	AvdArraySubDirectorios [6]int64
	AvdApDetalleDirectorio [5]int64
	AvdApIndirecto         int64
	AvdPropietario         [16]byte
}

// DetalleDirectorio ,
type DetalleDirectorio struct {
	DdArrayArchivosTXT [5]DetalleDirectorioArr
	DdApIndirecto      int64
}

// DetalleDirectorioArr ,
type DetalleDirectorioArr struct {
	DdNombreArchivoTXT  [16]byte
	DdApInodo           int64
	DdFechaCreacion     [20]byte
	DdFechaModificacion [20]byte
}

// Inodos ,
type Inodos struct {
	iNumeroInodo            int64
	iTamanioArchivoTXT      int64
	iNumeroBloquesAsignados int64
	iApArrayBloques         [4]int64
	iApIndirecto            int64
	iPropietario            [16]byte
}

type bloque struct {
	bInformacionArchivo [25]byte
}

// LogBitacora ,
type LogBitacora struct {
	LogTipoOperacion [20]byte
	LogTipo          int64    // 0 si es archivo, 1 si es directorio
	LogNombre        [20]byte //nombre archivo o directorio
	LogContenido     [15]byte
	LogFecha         [20]byte
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

	var bandera bool = false // me ayudara para que despues de -> pueda concatenar negativos -12, porque concatena ->- y aparte el numero

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
			} else if examinarAsci == 95 { // _
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
				caracter = caracter + string(examinar) //cancatena
				analizador2(caracter)                  //ingresa en lista, en este caso guardara cuando sea ->
				caracter = ""                          //limpiara
				estado = 5                             // y mandara al estado 5 (donde verifica si vendra numero negativo) ej. ->-23
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
		case 5:
			//cuando 'bandera' sea falso entrara aca
			//caracter vendra vacio ""
			//entonces si viene - va concatenar
			//si viene numero despues de guion va concatenar y activar el 'bandera'
			//entonces cuando venga otro - no lo va concatenar porque 'bandera' es true y entrara en el else
			if examinarAsci == 45 && bandera == false { // -
				caracter = caracter + string(examinar)
				estado = 5
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				bandera = true
				estado = 5
			} else {
				if strings.Compare(caracter, "") != 0 { //esto es porque: viene vacio de estado 2, y como viene una letra, entra aca e ingresa un vacio a la lista
					analizador2(caracter)
				}
				caracter = ""
				bandera = false
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
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "mount") == 0 {
			mountComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "unmount") == 0 {
			unmountComando(i)
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
					fmt.Println("\n[ pause: presione 'enter' ]")
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

	fmt.Println("\nDISCO CREADO CORRECTAMENTE:")
	fmt.Println("	Nombre: ", name)
	fmt.Println("	Ruta: ", ruta)
	fmt.Println("	Tamanio: ", size, " bytes")

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

	//si es delete
	if strings.Compare(deletee, "") != 0 { //si hay que eliminar una particion
		//si existe particion primaria o extendida con ese nombre
		if validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path, name) {
			eliminarParticionPrimariaExtendida(path, name)
			//si existe particion logica con ese nombre
		} else if validarSiExisteParticionLogicaConNombreEspecifico(path, name) {
			eliminarParticionLogica(path, name)
		} else {
			fmt.Print("\n[ ERROR: no exite particion con nombre: ", name, " ]")
		}

		//si es add
	} else if add != 0 { //agregar o quitar espacio de particion
		//si existe particion primaria o extendida con ese nombre
		if validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path, name) {
			//valida si se le puede agregar o quitar espacio
			if validaSiSeLePuedeaddEspacioEnPrimariaExtendida(path, name, unit, int64(add)) {
				//agrega o quita espacio en extendida o primaria
				addEspacioEnPrimariaExtendida(path, name, unit, int64(add))
			} else {
				fmt.Print("\n[ ERROR: no se puede agregar o quitar espacio en particion con nombre: ", name, " ]")
			}

			//si existe particion logica con ese nombre
		} else if validarSiExisteParticionLogicaConNombreEspecifico(path, name) {
			//valida si se le puede agregar o quitar espacio
			if validaSiSeLePuedeaddEspacioEnLogica(path, name, unit, int64(add)) {
				//agrega o quita espacio en logica
				addEspacioEnLogica(path, name, unit, int64(add))
			} else {
				fmt.Print("\n[ ERROR: no se puede agregar o quitar espacio en particion Logica con nombre: ", name, " ]")
			}
		} else {
			fmt.Print("\n[ ERROR: no exite particion con nombre: ", name, " ]")
		}

		//crear paarticion
	} else { //crea una particion
		agregarParticion(size, unit, path, typee, fit, name)
	}
}

func agregarParticion(size int64, unit string, path string, typee string, fit string, name string) {

	//esto es porque hay una validacion de: 4 primarias, o 3 primarias y una extendida
	if (strings.Compare(strings.ToLower(typee), "p") == 0) || (strings.Compare(strings.ToLower(typee), "e") == 0) {
		if validarLimiteDeParticionesEnDisco(path) { //si se puede agregar otra particion
			//si es primaria
			if strings.Compare(strings.ToLower(typee), "p") == 0 {
				//si el disco aun tiene espacio
				reposicionarPrimariaExtendida(path, size, unit)
				if validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida(path, size, unit) != -1 {
					//inserta particion primaria
					insertarParticionPrimaria(path, size, typee, fit, name, unit)
				} else {
					fmt.Print("\n[ ERROR: no hay espacio para agregar la particion primaria: ", name, " ]")
				}
				//si es extendida
			} else if strings.Compare(strings.ToLower(typee), "e") == 0 {
				//si aun no existe una extendida
				if validarSiExisteParticionExtendida(path) == false {
					//si el disco aun tiene espacio
					reposicionarPrimariaExtendida(path, size, unit)
					if validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida(path, size, unit) != -1 {
						//inserta particion extendida
						insertarParticionExtendida(path, size, typee, fit, name, unit)
					} else {
						fmt.Print("\n[ ERROR: no hay espacio para agregar la particion extendida: ", name, " ]")
					}

				} else {
					fmt.Print("\n[ ERROR: ya existe particion extendida en el disco, no se puede agregar: ", name, " ]")
				}
			}
		} else {
			fmt.Print("\n[ ERROR: Ya alcanzo el limite de de particiones en el disco: ]")
		}
		//si es logica
	} else if strings.Compare(strings.ToLower(typee), "l") == 0 {
		//si existe particion extendida
		if validarSiExisteParticionExtendida(path) {
			//si hay espacio dentro de la extendida
			if validarSiHayEspacioEnAlgunaPosicionLogica(path, size, unit) != -1 {
				//inserta particion logica
				reposicionarLogicas(path, size, unit)
				insertarParticionLogica(path, size, typee, fit, name, unit)
			} else {
				fmt.Print("\n[ ERROR: no hay espacio para agregar la particion logica: ", name, " ]")
			}
		} else {
			fmt.Print("\n[ ERROR: no existe particion extendida en el disco, no se puede agregar: ", name, " ]")
		}
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
//NO FUNCIONAL
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

	//espacio disponible = (tamanio disco) - (espacio actual de todas las particiones) - (lo que ocupa el mbr)
	var espacioDisponible int64 = int64(m.Tamanio) - contadorSize - int64(size)

	//si hay espacio aun
	if sizeParticion <= espacioDisponible {
		return true //retorna que hay espacio
	}

	return false
}

//valida si hay espacio, en el lugar que encuentra vacio
//si la posicion 2 esta vacia por ejemplo, valida que quepa ahi la particion
func validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida(path string, sizeParticion int64, unit string) int {
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

	//recorro para sumar todos los byte de las particiones
	for x := 0; x < 4; x++ {
		actual := misParticiones[x]
		//si encuentra una vacia
		if actual.Tamanio == 0 {

			//si es la posicion 0
			if x == 0 {
				//buscando particion siguiente
				posSiguiente := -1
				for i := x + 1; i < 4; i++ { //empieza a buscar una despues de la que se va eliminar
					//si encuantra despues una particion, guardo posicion donde la encuentra
					if misParticiones[i].Tamanio != 0 {
						posSiguiente = i
						break
					}
				}

				//CASO 1: TENGA SIGUIENTE
				if posSiguiente != -1 {
					//star de siguiente
					starSiguiente := misParticiones[posSiguiente].Start
					// espacio = star siguiente - tamanio MBR
					espacioDisponible := starSiguiente - int64(size)
					if espacioDisponible >= sizeParticion {
						return 0
					}

					//CASO 2: NO TENGA SIGUIENTE
				} else if posSiguiente == -1 {
					//tamanio disco
					tamDisco := m.Tamanio
					//espacio = tamanio disco - MBR
					espacioDisponible := tamDisco - int64(size)
					if espacioDisponible >= sizeParticion {
						return 0
					}
				}

				//si es la ultima posicion
			} else if x == 3 {
				//buscando particion anterior
				posAnterior := -1
				for i := x - 1; i > -1; i-- { //empieza a buscar una antes
					//si encuantra antes una particion, guardo posicion donde la encuentra
					if misParticiones[i].Tamanio != 0 {
						posAnterior = i
						break
					}
				}

				//CASO 1: TENGA ANTERIOR
				if posAnterior != -1 {
					starAnterior := misParticiones[posAnterior].Start
					tamAnterior := misParticiones[posAnterior].Tamanio
					//espacio = disco - (star anterior + tamanio anterior)
					espacioDisponible := m.Tamanio - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}

					//CASO 2: NO TIENE ANTERIOR
				} else if posAnterior == -1 {
					//no retornara nada, porque si hubiera espacio antes, ahi se hubiera insertado
				}

				//es 1 o 2
			} else {
				//buscando particion siguiente
				posSiguiente := -1
				for i := x + 1; i < 4; i++ { //empieza a buscar una despues de la que se va eliminar
					//si encuantra despues una particion, guardo posicion donde la encuentra
					if misParticiones[i].Tamanio != 0 {
						posSiguiente = i
						break
					}
				}

				//buscando particion anterior
				posAnterior := -1
				for i := x - 1; i > -1; i-- { //empieza a buscar una antes
					//si encuantra antes una particion, guardo posicion donde la encuentra
					if misParticiones[i].Tamanio != 0 {
						posAnterior = i
						break
					}
				}

				//CASO 1: TENGA ANTERIOR Y NO SIGUIENTE
				if posAnterior != -1 && posSiguiente == -1 {
					starAnterior := misParticiones[posAnterior].Start
					tamAnterior := misParticiones[posAnterior].Tamanio
					//espacio = disco - (star anterior + tamanio anterior)
					espacioDisponible := m.Tamanio - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}

					//CASO 2: TENGA SIGUIENTE Y NO ANTERIOR
				} else if posSiguiente != -1 && posAnterior == -1 {
					// no retorna nada porque sino se hubiera insertado en alguna posicion anterior

					//CASO 3: TIENE SIGUIENTE Y TIENE ANTERIOR
				} else if posSiguiente != -1 && posAnterior != -1 {
					starSiguiente := misParticiones[posSiguiente].Start
					starAnterior := misParticiones[posAnterior].Start
					tamAnterior := misParticiones[posAnterior].Tamanio
					//espacio = star siguiente - (star anterior + tamanio anterior)
					espacioDisponible := starSiguiente - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}
				}
			}
		}
	}

	return -1
}

//inserta particion primaria en el disco
func insertarParticionPrimaria(path string, sizePart int64, typee string, fit string, name string, unit string) {

	//el metodo 'validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida' hace su propia conversion a bytes, por eso se le envia la original
	sizeAntesDeCombertir := sizePart

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

	//para ver la posicion vacia
	contador := validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida(path, sizeAntesDeCombertir, unit)

	/*
		//recorro para ver cuual esta vacia
		for i := 0; i < 4; i++ {
			actual := misParticiones[i]
			if actual.Tamanio == 0 {
				contador = i
				break
			}
		}*/

	//se inserta despues del MBR
	if contador == 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)
		particionPrimariaNew.Start = int64(size) + 1

		//inserto particion
		misParticiones[contador] = particionPrimariaNew

		//pueden ser en la posicion 1, 2, 3
	} else if contador > 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)

		//Donde empieza? empieza donde termina la particion anterior
		inicioPartAnt := misParticiones[contador-1].Start //byte donde inicia la particion anterior
		tamPartAnt := misParticiones[contador-1].Tamanio  //tamanio de la particion anterior
		finPartAnt := inicioPartAnt + tamPartAnt          //byte donde finaliza la particion anterior
		particionPrimariaNew.Start = finPartAnt + 1       //aqui empieza la nueva particion

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
	}

	fmt.Println("\nINSERTO PRIMARIA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

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

//valida si hay una particion extendida en el disco
func validarSiExisteParticionExtendida(path string) bool {
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

	existe := false

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			existe = true
			break
		}
	}

	//si ya existe una extendida
	if existe {
		return true
	}

	return false
}

//inserta particion extendida en el disco
func insertarParticionExtendida(path string, sizePart int64, typee string, fit string, name string, unit string) {
	//el metodo 'validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida' hace su propia conversion a bytes, por eso se le envia la original
	sizeAntesDeCombertir := sizePart

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

	//para ver la posicion vacia
	contador := validarSiHayEspacioEnAlgunaPosicionPrimariaExtendida(path, sizeAntesDeCombertir, unit)

	/*
		//recorro para ver cuual esta vacia
		for i := 0; i < 4; i++ {
			actual := misParticiones[i]
			if actual.Tamanio == 0 {
				contador = i
				break
			}
		}*/

	//se inserta despues del MBR
	if contador == 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)
		particionPrimariaNew.Start = int64(size) + 1

		//inserto particion
		misParticiones[contador] = particionPrimariaNew

		//pueden ser en la posicion 1, 2, 3
	} else if contador > 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)

		//Donde empieza? empieza donde termina la particion anterior
		inicioPartAnt := misParticiones[contador-1].Start //byte donde inicia la particion anterior
		tamPartAnt := misParticiones[contador-1].Tamanio  //tamanio de la particion anterior
		finPartAnt := inicioPartAnt + tamPartAnt          //byte donde finaliza la particion anterior
		particionPrimariaNew.Start = finPartAnt + 1       //aqui empieza la nueva particion

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
	}

	fmt.Println("\nINSERTO EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

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

//esta funcion crea un espacio entre dos particiones, si la nueva particion cabe entre ellas
func reposicionarPrimariaExtendida(path string, sizePart int64, unit string) {

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

	//recorera hasta una menos del tamanio de array
	for i := 0; i < len(misParticiones)-1; i++ {

		//REPOSICIONARA CUANDO ALLA ESPACIO ENTRE ULTIMO Y EL DISCO
		if i+1 == len(misParticiones)-1 {
			//si hay una particion en la ultima posicion
			if misParticiones[i+1].Tamanio != 0 {
				starActual := misParticiones[i+1].Start
				tamActual := misParticiones[i+1].Tamanio
				tamDisco := m.Tamanio
				//espacio = tamDisco - (star actual + tam actual)
				espacio := tamDisco - (starActual + tamActual)
				fmt.Println(espacio)
				//si hay espacio
				if espacio >= sizePart {
					posAnterior := -1
					for x := i; x > -1; x-- { //empieza a buscar una antes (i-1)
						//si encuantra antes un espacio vacio
						if misParticiones[x].Tamanio == 0 {
							posAnterior = x
							break
						}
					}

					//quiere decir que encontro un espacio
					if posAnterior != -1 {
						for y := 0; y < i+1; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
							//si esta en un espacio vacio
							if misParticiones[y].Tamanio == 0 {
								//si en la siguiente hay una particion
								if misParticiones[y+1].Tamanio != 0 {
									//reposicionamos posicion
									vacio := NodoParticion{}
									misParticiones[y] = misParticiones[y+1]
									misParticiones[y+1] = vacio
								}
							}
						}
					}

				}
			}
		}

		//REPOSICIONARA SIEMPRE Y CUANDO ALLA ESPACIO ENTRE DOS JUNTAS
		//si encuentra una particion
		if misParticiones[i].Tamanio != 0 {
			//si en la siguiente posicion hay otra particion
			if misParticiones[i+1].Tamanio != 0 {
				//calculo espacio entre ambas
				starActual := misParticiones[i].Start
				tamActual := misParticiones[i].Tamanio
				starSiguiente := misParticiones[i+1].Start
				//espacio = star siguiente - (star actual + tam actual)
				espacio := starSiguiente - (starActual + tamActual)
				//si hay espacio entre ambas
				if espacio >= sizePart {

					//quiere decir que el espacio esta entre la antepenultima y ultima posicion
					if (i + 1) == len(misParticiones)-1 {
						//buscando particion vacia anterior
						posAnterior := -1
						for x := i - 1; x > -1; x-- { //empieza a buscar una antes (i-1)
							//si encuantra antes un espacio vacio
							if misParticiones[x].Tamanio == 0 {
								posAnterior = x
								break
							}
						}

						//quiere decir que encontro un espacio
						if posAnterior != -1 {
							for y := 0; y < i; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
								//si esta en un espacio vacio
								if misParticiones[y].Tamanio == 0 {
									//si en la siguiente hay una particion
									if misParticiones[y+1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticion{}
										misParticiones[y] = misParticiones[y+1]
										misParticiones[y+1] = vacio
									}
								}
							}
						}

						//quiere decir que el espacio entrea en la primera y segunda posicion
					} else if i == 0 {
						//buscando particion vacia siguiente
						posSiguiente := -1
						for x := i + 2; x < len(misParticiones); x++ { //empieza a buscar una despues, de las dos donde se encontro espacio
							//si encuantra un espacio
							if misParticiones[x].Tamanio == 0 {
								posSiguiente = x
								break
							}
						}

						//si encontro un espacio
						if posSiguiente != -1 {
							for y := len(misParticiones) - 1; y > i+1; y-- { //empieza desde el ultimo
								//si encuentra una vacia
								if misParticiones[y].Tamanio == 0 {
									//si la anterior no esta vacia
									if misParticiones[y-1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticion{}
										misParticiones[y] = misParticiones[y-1]
										misParticiones[y-1] = vacio
									}
								}
							}
						}
					} else {
						//buscando particion vacia siguiente
						posSiguiente := -1
						for x := i + 2; x < len(misParticiones); x++ { //empieza a buscar una despues, de las dos donde se encontro espacio
							//si encuantra un espacio
							if misParticiones[x].Tamanio == 0 {
								posSiguiente = x
								break
							}
						}

						//buscando particion vacia anterior
						posAnterior := -1
						for x := i - 1; x > -1; x-- { //empieza a buscar una antes (i-1)
							//si encuantra antes un espacio vacio
							if misParticiones[x].Tamanio == 0 {
								posAnterior = x
								break
							}
						}

						//si hay una posicion vacia despues
						if posSiguiente != -1 {
							for y := len(misParticiones) - 1; y > i+1; y-- { //empieza desde el ultimo
								//si encuentra una vacia
								if misParticiones[y].Tamanio == 0 {
									//si la anterior no esta vacia
									if misParticiones[y-1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticion{}
										misParticiones[y] = misParticiones[y-1]
										misParticiones[y-1] = vacio
									}
								}
							}

							//si hay una posicion vacia antes
						} else if posAnterior != -1 {
							for y := 0; y < i; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
								//si esta en un espacio vacio
								if misParticiones[y].Tamanio == 0 {
									//si en la siguiente hay una particion
									if misParticiones[y+1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticion{}
										misParticiones[y] = misParticiones[y+1]
										misParticiones[y+1] = vacio
									}
								}
							}
						}
					}
				}
			}
		}
	}

	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())
}

//valida si hay espacio en la particion extendida, dentro del disco
//NO FUNCIONAL
func validarQueTengaEspacioParticionExtendida(path string, sizeParticion int64, unit string) bool {

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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	//sera la sumatoria de byte de todas las particiones primarias
	var contadorSize int64 = 0

	//recorro para sumar todos los byte de las particiones
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]
		if actual.Tamanio != 0 {
			contadorSize = contadorSize + int64(actual.Tamanio)
		}
	}

	//espacio disponible = (tamanio particion extendida) - (espacio actual de todas las particiones logicas)
	var espacioDisponible int64 = int64(misParticiones[posicionExtendida].Tamanio) - contadorSize

	//si hay espacio aun
	if sizeParticion <= espacioDisponible {
		return true //retorna que hay espacio
	}

	return false
}

func validarSiHayEspacioEnAlgunaPosicionLogica(path string, sizeParticion int64, unit string) int {

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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	for x := 0; x < len(misParticionesLogicas); x++ {
		actual := misParticionesLogicas[x]

		//si encuentra una vacia
		if actual.Tamanio == 0 {

			//si es la posicion 0
			if x == 0 {
				//buscando particion siguiente
				posSiguiente := -1
				for i := x + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues de la que se va eliminar
					//si encuantra despues una particion, guardo posicion donde la encuentra
					if misParticionesLogicas[i].Tamanio != 0 {
						posSiguiente = i
						break
					}
				}

				//CASO 1: TENGA SIGUIENTE
				if posSiguiente != -1 {
					//star de siguiente
					starSiguiente := misParticionesLogicas[posSiguiente].Start
					// espacio = star siguiente - star extendida
					espacioDisponible := starSiguiente - misParticiones[posicionExtendida].Start
					if espacioDisponible >= sizeParticion {
						return 0
					}

					//CASO 2: NO TENGA SIGUIENTE
				} else if posSiguiente == -1 {
					//quiere decir que esta vacia la extend
					//tamanio extendida
					tamExtend := misParticiones[posicionExtendida].Tamanio
					if tamExtend >= sizeParticion {
						return 0
					}
				}

				//si es la ultima posicion
			} else if x == len(misParticionesLogicas)-1 {
				//buscando particion anterior
				posAnterior := -1
				for i := x - 1; i > -1; i-- { //empieza a buscar una antes
					//si encuantra antes una particion, guardo posicion donde la encuentra
					if misParticionesLogicas[i].Tamanio != 0 {
						posAnterior = i
						break
					}
				}

				//CASO 1: TENGA ANTERIOR
				if posAnterior != -1 {
					starAnterior := misParticionesLogicas[posAnterior].Start
					tamAnterior := misParticionesLogicas[posAnterior].Tamanio
					starExtend := misParticiones[posicionExtendida].Start
					tamExtend := misParticiones[posicionExtendida].Tamanio
					//espacio = tamanio extend - (star anterior + tamanio anterior)
					espacioDisponible := (starExtend + tamExtend) - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}

					//CASO 2: NO TIENE ANTERIOR
				} else if posAnterior == -1 {
					//no retornara nada, porque si hubiera espacio antes, ahi se hubiera insertado
				}

				//no es ni 0 ni el ultimo
			} else {

				//buscando particion siguiente
				posSiguiente := -1
				for i := x + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues de la que se va eliminar
					//si encuantra despues una particion, guardo posicion donde la encuentra
					if misParticionesLogicas[i].Tamanio != 0 {
						posSiguiente = i
						break
					}
				}

				//buscando particion anterior
				posAnterior := -1
				for i := x - 1; i > -1; i-- { //empieza a buscar una antes
					//si encuantra antes una particion, guardo posicion donde la encuentra
					if misParticionesLogicas[i].Tamanio != 0 {
						posAnterior = i
						break
					}
				}

				//CASO 1: TENGA ANTERIOR Y NO SIGUIENTE
				if posAnterior != -1 && posSiguiente == -1 {
					starAnterior := misParticionesLogicas[posAnterior].Start
					tamAnterior := misParticionesLogicas[posAnterior].Tamanio
					starExtend := misParticiones[posicionExtendida].Start
					tamExtend := misParticiones[posicionExtendida].Tamanio
					//espacio = tamanio extend - (star anterior + tamanio anterior)
					espacioDisponible := (starExtend + tamExtend) - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}

					//CASO 2: TENGA SIGUIENTE Y NO ANTERIOR
				} else if posSiguiente != -1 && posAnterior == -1 {
					// no retorna nada porque sino se hubiera insertado en alguna posicion anterior

					//CASO 3: TIENE SIGUIENTE Y TIENE ANTERIOR
				} else if posSiguiente != -1 && posAnterior != -1 {
					starSiguiente := misParticionesLogicas[posSiguiente].Start
					starAnterior := misParticionesLogicas[posAnterior].Start
					tamAnterior := misParticionesLogicas[posAnterior].Tamanio
					//espacio = star siguiente - (star anterior + tamanio anterior)
					espacioDisponible := starSiguiente - (starAnterior + tamAnterior)
					if espacioDisponible >= sizeParticion {
						return x
					}
				}

			}
		}

	}

	return -1
}

//inserta particion logica, dentro de la extendida
func insertarParticionLogica(path string, sizePart int64, typee string, fit string, name string, unit string) {
	//el metodo 'validarSiHayEspacioEnAlgunaPosicionLogica hace su propia conversion a bytes, por eso se le envia la original
	sizeAntesDeCombertir := sizePart

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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a las particiones logicas, quee estan dentro de la extendida
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	//para ver la posicion logica vacia
	posicionLogicaVacia := validarSiHayEspacioEnAlgunaPosicionLogica(path, sizeAntesDeCombertir, unit)

	/*//recorro para ver cuual esta vacia
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]
		if actual.Tamanio == 0 {
			posicionLogicaVacia = i
			break
		}
	}*/

	//se inserta despues del MBR
	if posicionLogicaVacia == 0 {

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogicaVacia + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar desde la posicion 1, y que insertara en la posicion 0
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: no tenga siguiente
		if posSiguiente == -1 {

			//creo particion primaria
			particionLogicaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionLogicaNew.Name[:], name)
			particionLogicaNew.Tamanio = sizePart
			particionLogicaNew.Estado = 1
			particionLogicaNew.TipoParticion = typee[0]
			copy(particionLogicaNew.TipoAjuste[:], fit)
			particionLogicaNew.Start = int64(misParticiones[posicionExtendida].Start) + 1 //inicia donde inicia la extend
			particionLogicaNew.Next = int64(-1)

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionLogicaNew
			fmt.Println("Inicio: ", name, " : ", particionLogicaNew.Start)

			//CASO 2: tenga siguiente
		} else if posSiguiente != -1 {

			//creo particion primaria
			particionLogicaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionLogicaNew.Name[:], name)
			particionLogicaNew.Tamanio = sizePart
			particionLogicaNew.Estado = 1
			particionLogicaNew.TipoParticion = typee[0]
			copy(particionLogicaNew.TipoAjuste[:], fit)
			particionLogicaNew.Start = int64(misParticiones[posicionExtendida].Start) + 1 //inicia donde inicia la extend
			particionLogicaNew.Next = misParticionesLogicas[posSiguiente].Start

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionLogicaNew
			fmt.Println("Inicio: ", name, " : ", particionLogicaNew.Start)

		}

		//pueden ser en la posicion 1, 2, 3
	} else if posicionLogicaVacia > 0 {

		//buscando particion anterior
		posAnterior := -1
		for i := posicionLogicaVacia - 1; i > -1; i-- { //empieza a buscar una antes de la que se va eliminar
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogicaVacia + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues de la que se va eliminar
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: tenga anterior y no tenga siguiente
		if (posAnterior != -1) && (posSiguiente == -1) {

			//creo particion primaria
			particionPrimariaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionPrimariaNew.Name[:], name)
			particionPrimariaNew.Tamanio = sizePart
			particionPrimariaNew.Estado = 1
			particionPrimariaNew.TipoParticion = typee[0]
			copy(particionPrimariaNew.TipoAjuste[:], fit)

			//Donde empieza? empieza donde termina la particion anterior
			inicioPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Start //byte donde inicia la particion anterior
			tamPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Tamanio  //tamanio de la particion anterior
			finPartAnt := inicioPartAnt + tamPartAnt                            //byte donde finaliza la particion anterior
			particionPrimariaNew.Start = finPartAnt + 1                         //aqui empieza la nueva particion

			//next anterior = star nuevo
			//next nuevo = -1
			misParticionesLogicas[posAnterior].Next = particionPrimariaNew.Start
			particionPrimariaNew.Next = int64(-1)

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionPrimariaNew
			fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)

			//CASO 2: tenga anterior y siguiente
		} else if (posAnterior != -1) && (posSiguiente != -1) {

			//creo particion primaria
			particionPrimariaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionPrimariaNew.Name[:], name)
			particionPrimariaNew.Tamanio = sizePart
			particionPrimariaNew.Estado = 1
			particionPrimariaNew.TipoParticion = typee[0]
			copy(particionPrimariaNew.TipoAjuste[:], fit)

			//Donde empieza? empieza donde termina la particion anterior
			inicioPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Start //byte donde inicia la particion anterior
			tamPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Tamanio  //tamanio de la particion anterior
			finPartAnt := inicioPartAnt + tamPartAnt                            //byte donde finaliza la particion anterior
			particionPrimariaNew.Start = finPartAnt + 1                         //aqui empieza la nueva particion

			//next nuevo = next anterior
			//next anterior = star nuevo
			particionPrimariaNew.Next = misParticionesLogicas[posAnterior].Next
			misParticionesLogicas[posAnterior].Next = particionPrimariaNew.Start

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionPrimariaNew
			fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)
		}

	}

	fmt.Println("\nINSERTO LOGICA:")
	for i := 0; i < len(misParticionesLogicas); i++ {
		fmt.Println("	arr pos ", i, " Tamanio : ", misParticionesLogicas[i].Tamanio, " Star: ", misParticionesLogicas[i].Start, " Next: ", misParticionesLogicas[i].Next, " Tipo: ", string(misParticionesLogicas[i].TipoParticion))
	}

	//las particiones logicas actuales se encuentran en 'misParticiones[posicionExtendida].ParticionesLogicas'
	//cuando se crea una nueva particion logica se agregan a 'misParticionesPrimarias'
	//entonces 'misParticionesPrimarias' tienen las actuales, mas la nueva que se inserto
	//por eso se iguala de nuevo, para que 'misParticiones[posicionExtendida].ParticionesLogicas', e guarden particiones ya actualizadas
	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())
}

//validar si existe particion primaria o extendida dado nombre (para delete)
func validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path string, name string) bool {
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

	existe := false

	//recorro para ver si existe nombre
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			existe = true
			break
		}
	}

	//si existe particion con ese nombre
	if existe {
		return true
	}

	return false

}

//esta funcion crea un espacio entre dos particiones, si la nueva particion cabe entre ellas
func reposicionarLogicas(path string, sizeParticion int64, unit string) {

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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	//recorrera hasta una menos del tamanio de array
	for i := 0; i < len(misParticionesLogicas)-1; i++ {

		//REPOSICIONARA CUANDO ALLA ESPACIO ENTRE ULTIMO Y EL DISCO
		if i+1 == len(misParticionesLogicas)-1 {
			//si hay una particion en la ultima posicion
			if misParticionesLogicas[i+1].Tamanio != 0 {
				starActual := misParticionesLogicas[i+1].Start
				tamActual := misParticionesLogicas[i+1].Tamanio
				starExtend := misParticiones[posicionExtendida].Start
				tamExtend := misParticiones[posicionExtendida].Tamanio
				//espacio = tamDisco - (star actual + tam actual)
				espacio := (starExtend + tamExtend) - (starActual + tamActual)
				//si hay espacio
				if espacio >= sizeParticion {
					posAnterior := -1
					for x := i; x > -1; x-- { //empieza a buscar una antes (i-1)
						//si encuantra antes un espacio vacio
						if misParticionesLogicas[x].Tamanio == 0 {
							posAnterior = x
							break
						}
					}

					//quiere decir que encontro un espacio
					if posAnterior != -1 {
						for y := 0; y < i+1; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
							//si esta en un espacio vacio
							if misParticionesLogicas[y].Tamanio == 0 {
								//si en la siguiente hay una particion
								if misParticionesLogicas[y+1].Tamanio != 0 {
									//reposicionamos posicion
									vacio := NodoParticionLogica{}
									misParticionesLogicas[y] = misParticionesLogicas[y+1]
									misParticionesLogicas[y+1] = vacio
								}
							}
						}
					}

				}
			}
		}

		//REPOSICIONARA SIEMPRE Y CUANDO ALLA ESPACIO ENTRE DOS JUNTAS
		//si encuentra una particion logica
		if misParticionesLogicas[i].Tamanio != 0 {
			//si en la siguiente posicion hay una posicion logica
			if misParticionesLogicas[i+1].Tamanio != 0 {
				//calculo espacio entre ambas
				starActual := misParticionesLogicas[i].Start
				tamActual := misParticionesLogicas[i].Tamanio
				starSiguiente := misParticionesLogicas[i+1].Start
				//espacio = star siguiente - (star actual + tam actual)
				espacio := starSiguiente - (starActual + tamActual)
				//si hay espacio entre ambas
				if espacio >= sizeParticion {

					//quiere decir que el espacion esta entre la antepenultima y ultima particion
					if (i + 1) == len(misParticionesLogicas)-1 {
						//buscando particion vacia anterior
						posAnterior := -1
						for x := i - 1; x > -1; x-- { //empieza a buscar una antes (i-1)
							//si encuantra antes un espacio vacio
							if misParticionesLogicas[x].Tamanio == 0 {
								posAnterior = x
								break
							}
						}

						//quiere decir que encontro un espacio
						if posAnterior != -1 {
							for y := 0; y < i; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
								//si esta en un espacio vacio
								if misParticionesLogicas[y].Tamanio == 0 {
									//si en la siguiente hay una particion
									if misParticionesLogicas[y+1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticionLogica{}
										misParticionesLogicas[y] = misParticionesLogicas[y+1]
										misParticionesLogicas[y+1] = vacio
									}
								}
							}
						}

						//quiere decir que el espacio entrea en la primera y segunda posicion
					} else if i == 0 {
						//buscando particion vacia siguiente
						posSiguiente := -1
						for x := i + 2; x < len(misParticionesLogicas); x++ { //empieza a buscar una despues, de las dos donde se encontro espacio
							//si encuantra un espacio
							if misParticionesLogicas[x].Tamanio == 0 {
								posSiguiente = x
								break
							}
						}

						//si encontro un espacio
						if posSiguiente != -1 {
							for y := len(misParticionesLogicas) - 1; y > i+1; y-- { //empieza desde el ultimo
								//si encuentra una vacia
								if misParticionesLogicas[y].Tamanio == 0 {
									//si la anterior no esta vacia
									if misParticionesLogicas[y-1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticionLogica{}
										misParticionesLogicas[y] = misParticionesLogicas[y-1]
										misParticionesLogicas[y-1] = vacio
									}
								}
							}
						}
					} else {
						//buscando particion vacia siguiente
						posSiguiente := -1
						for x := i + 2; x < len(misParticionesLogicas); x++ { //empieza a buscar una despues, de las dos donde se encontro espacio
							//si encuantra un espacio
							if misParticionesLogicas[x].Tamanio == 0 {
								posSiguiente = x
								break
							}
						}

						//buscando particion vacia anterior
						posAnterior := -1
						for x := i - 1; x > -1; x-- { //empieza a buscar una antes (i-1)
							//si encuantra antes un espacio vacio
							if misParticionesLogicas[x].Tamanio == 0 {
								posAnterior = x
								break
							}
						}

						//si hay una posicion vacia despues
						if posSiguiente != -1 {
							for y := len(misParticionesLogicas) - 1; y > i+1; y-- { //empieza desde el ultimo
								//si encuentra una vacia
								if misParticionesLogicas[y].Tamanio == 0 {
									//si la anterior no esta vacia
									if misParticionesLogicas[y-1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticionLogica{}
										misParticionesLogicas[y] = misParticionesLogicas[y-1]
										misParticionesLogicas[y-1] = vacio
									}
								}
							}

							//si hay una posicion vacia antes
						} else if posAnterior != -1 {
							for y := 0; y < i; y++ { //ira movimiento hasta la posicion de la particion donde encontro espacio ( < i )
								//si esta en un espacio vacio
								if misParticionesLogicas[y].Tamanio == 0 {
									//si en la siguiente hay una particion
									if misParticionesLogicas[y+1].Tamanio != 0 {
										//reposicionamos posicion
										vacio := NodoParticionLogica{}
										misParticionesLogicas[y] = misParticionesLogicas[y+1]
										misParticionesLogicas[y+1] = vacio
									}
								}
							}
						}
					}

				}
			}
		}

	}

	/*fmt.Println("\nREPOSICION LOGICA:")
	for i := 0; i < len(misParticionesLogicas); i++ {
		fmt.Println("	arr pos ", i, " Tamanio : ", misParticionesLogicas[i].Tamanio, " Star: ", misParticionesLogicas[i].Start, " Next: ", misParticionesLogicas[i].Next, " Tipo: ", string(misParticionesLogicas[i].TipoParticion))
	}*/

	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//eliminar particion primaria o extendida
func eliminarParticionPrimariaExtendida(path string, name string) {
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

	posicionParticion := 0

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//creo una particion vacia
	particionNew := NodoParticion{}

	//inserto particion vacia en la posicion a eliminar
	misParticiones[posicionParticion] = particionNew

	fmt.Println("\nDELETE PRIMARIA O EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se elimino una particion se elimino de 'misPartiiones'
	//entonces 'misParticiones' tiene las actuales, con la que se acaba de liminar
	//por eso se iguala de nuevo, para que em 'm.Particiones' ya no aparezca la que se elimino
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//func validar existe particion logica dado nombre (para delete)
func validarSiExisteParticionLogicaConNombreEspecifico(path string, name string) bool {

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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	existe := false

	//recorro para ver si existe nombre
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			existe = true
			break
		}
	}

	//si existe logica con ese nombre
	if existe {
		return true //retorna que hay espacio
	}

	return false
}

//elimina particion logica
func eliminarParticionLogica(path string, name string) {
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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	posicionLogica := 0

	//recorro para ver si existe nombre
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionLogica = i
			break
		}
	}

	//creo particion logica vacia
	particionLogicaNew := NodoParticionLogica{}

	//redireccionando next de particiones logicas
	//si la que se va eliminar esta en la primera posicion
	if posicionLogica == 0 {
		//solo se elimina
		misParticionesLogicas[posicionLogica] = particionLogicaNew

		//si la particiona eliminar esta en la ultima posicion
	} else if posicionLogica == len(misParticionesLogicas)-1 {
		//buscando particion anterior
		posAnterior := -1
		for i := len(misParticionesLogicas) - 2; i > -1; i-- {
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}
		//quiere decir que si encontro una particion antes
		if posAnterior != -1 {
			//la particion que esta antes ahora apunta a -1
			misParticionesLogicas[posAnterior].Next = int64(-1)
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//quiere decir que solo existe una particion, y es en la ultima
		} else {
			//solo se elimina
			misParticionesLogicas[posicionLogica] = particionLogicaNew
		}
		//si la particiona eliminar no es la primera ni la ultima
	} else {
		//buscando particion anterior
		posAnterior := -1
		for i := posicionLogica - 1; i > -1; i-- { //empieza a buscar una antes de la que se va eliminar
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogica + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues de la que se va eliminar
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: que tenga siguiente pero no anterior
		if (posSiguiente != -1) && (posAnterior == -1) {
			//solo se elimna
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 2: que tenga anterior pero no siguiente
		} else if (posSiguiente == -1) && (posAnterior != -1) {
			//next de parte anterior igual a -1
			misParticionesLogicas[posAnterior].Next = -1
			//elimino
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 3: que no tenga siguiente ni anterior
		} else if (posSiguiente == -1) && (posAnterior == -1) {
			//solo se elimna
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 4: que tenga siguiente y anterior
		} else if (posSiguiente != -1) && (posAnterior != -1) {
			//next de parte interior=next de parte a eliminar
			misParticionesLogicas[posAnterior].Next = misParticionesLogicas[posicionLogica].Next
			//elimino
			misParticionesLogicas[posicionLogica] = particionLogicaNew
		}

	}

	fmt.Println("\nDELETE LOGICA:")
	for i := 0; i < len(misParticionesLogicas); i++ {
		fmt.Println("	arr pos ", i, " Tamanio : ", misParticionesLogicas[i].Tamanio, " Star: ", misParticionesLogicas[i].Start, " Next: ", misParticionesLogicas[i].Next, " Tipo: ", string(misParticionesLogicas[i].TipoParticion))
	}

	//las particiones logicas actuales se encuentran en 'misParticiones[posicionExtendida].ParticionesLogicas'
	//cuando se elimina una particion logica se elimina de 'misParticionesPrimarias'
	//entonces 'misParticionesPrimarias' tienen las actuales, y se quito la que se elimino
	//por eso se iguala de nuevo, para que 'misParticiones[posicionExtendida].ParticionesLogicas', se actualice con la particion eliminada
	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//valida si se le puede agregar o quitar espacio en primaria o extendida
func validaSiSeLePuedeaddEspacioEnPrimariaExtendida(path string, name string, unit string, add int64) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
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

	posicionParticion := 0

	//recorro para ver en que posicion esta la particion con el name
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionParticion == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//si se puede agregar espacio
					return true
					//newTamanaio := tamanioParticion + add
					//misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionParticion == 3 {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el tamanio del disco
			tamDisco := m.Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticiones[posicionParticion].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := tamDisco - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				return true
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//star acutal
				starActual := misParticiones[posicionParticion].Start
				//tamanio actual
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					return true
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}
	}

	return false
}

//agrega o quita espacio en primaria o extendida
func addEspacioEnPrimariaExtendida(path string, name string, unit string, add int64) {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
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

	posicionParticion := 0

	//recorro para ver en que posicion esta la particion con el name
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionParticion == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//si se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionParticion == 3 {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el tamanio del disco
			tamDisco := m.Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticiones[posicionParticion].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := tamDisco - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				newTamanaio := tamanioParticion + add
				misParticiones[posicionParticion].Tamanio = newTamanaio
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//star acutal
				starActual := misParticiones[posicionParticion].Start
				//tamanio actual
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}
	}

	fmt.Println("\nSE ADD O DELETE ESPACIO EN PRIMARIA O EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se modifica el tamanio de una nueva particion se modifica en 'misPartiiones'
	//entonces 'misParticiones' tiene las anteriores, y la que se modifico ahora
	//por eso se iguala de nuevo, para que em 'm.Particiones' se guarden particiones ya actualizadas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

func validaSiSeLePuedeaddEspacioEnLogica(path string, name string, unit string, add int64) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	posicionLogica := 0

	//recorro para ver si existe nombre
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionLogica = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionLogica == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionLogica + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticionesLogicas[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticionesLogicas[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//si se puede agregar espacio
					return true
					//newTamanaio := tamanioParticion + add
					//misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el star de la extendida
				starExtend := misParticiones[posicionExtendida].Start
				//obtengo el tamanio de extendida
				tamExtend := misParticiones[posicionExtendida].Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionLogica == (len(misParticionesLogicas) - 1) {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el star de la extendida
			starExtend := misParticiones[posicionExtendida].Start
			//obtengo el tamanio de extendida
			tamExtend := misParticiones[posicionExtendida].Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticionesLogicas[posicionLogica].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				return true
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionLogica + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticionesLogicas[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticionesLogicas[posSiguiente].Start
				//star acutal
				starActual := misParticionesLogicas[posicionLogica].Start
				//tamanio actual
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					return true
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el star de la extendida
				starExtend := misParticiones[posicionExtendida].Start
				//obtengo el tamanio de extendida
				tamExtend := misParticiones[posicionExtendida].Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}
	}

	return false
}

func addEspacioEnLogica(path string, name string, unit string, add int64) {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
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

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	posicionLogica := 0

	//recorro para ver si existe nombre
	for i := 0; i < len(misParticionesLogicas); i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionLogica = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionLogica == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionLogica + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticionesLogicas[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticionesLogicas[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticionesLogicas[posicionLogica].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el star de la extendida
				starExtend := misParticiones[posicionExtendida].Start
				//obtengo el tamanio de extendida
				tamExtend := misParticiones[posicionExtendida].Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticionesLogicas[posicionLogica].Tamanio = newTamanaio
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticionesLogicas[posicionLogica].Tamanio = nuevoTamanio
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionLogica == (len(misParticionesLogicas) - 1) {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el star de la extendida
			starExtend := misParticiones[posicionExtendida].Start
			//obtengo el tamanio de extendida
			tamExtend := misParticiones[posicionExtendida].Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticionesLogicas[posicionLogica].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				newTamanaio := tamanioParticion + add
				misParticionesLogicas[posicionLogica].Tamanio = newTamanaio
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticionesLogicas[posicionLogica].Tamanio = nuevoTamanio
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionLogica + 1; i < len(misParticionesLogicas); i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticionesLogicas[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticionesLogicas[posSiguiente].Start
				//star acutal
				starActual := misParticionesLogicas[posicionLogica].Start
				//tamanio actual
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticionesLogicas[posicionLogica].Tamanio = newTamanaio
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el star de la extendida
				starExtend := misParticiones[posicionExtendida].Start
				//obtengo el tamanio de extendida
				tamExtend := misParticiones[posicionExtendida].Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticionesLogicas[posicionLogica].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := (starExtend + tamExtend) - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticionesLogicas[posicionLogica].Tamanio = newTamanaio
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticionesLogicas[posicionLogica].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				//se puede agregar espacio
				misParticionesLogicas[posicionLogica].Tamanio = nuevoTamanio
			}
		}
	}

	fmt.Println("\nADD O DELETE LOGICA:")
	for i := 0; i < len(misParticionesLogicas); i++ {
		fmt.Println("	arr pos ", i, " Tamanio : ", misParticionesLogicas[i].Tamanio, " Star: ", misParticionesLogicas[i].Start, " Next: ", misParticionesLogicas[i].Next, " Tipo: ", string(misParticionesLogicas[i].TipoParticion))
	}

	//las particiones logicas actuales se encuentran en 'misParticiones[posicionExtendida].ParticionesLogicas'
	//cuando se crea una nueva particion logica se agregan a 'misParticionesLogicas'
	//entonces 'misParticionesLogicas' tienen las actuales, mas la que se modifico tamanio
	//por eso se iguala de nuevo, para que 'misParticiones[posicionExtendida].ParticionesLogicas', se guarden particiones ya actualizadas
	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
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
	//no vienen parametros, mostrara particiones montadas
	if strings.Compare(path, "") == 0 && strings.Compare(path, "") == 0 {
		//muestra particiones montadas
		mostrarMount()
	} else {
		//si es particion primaria o extend
		if validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path, name) {
			generarIDMountYGuardar(path, name)
			fmt.Println("SE MONTO PARTICION:")
			fmt.Println("	Nombre: ", name)
			//si es particion logica
		} else if validarSiExisteParticionLogicaConNombreEspecifico(path, name) {
			generarIDMountYGuardar(path, name)
			fmt.Println("SE MONTO PARTICION:")
			fmt.Println("	Nombre: ", name)
		} else {
			fmt.Print("\n[ ERROR: no se puede montar particion con nombre: ", name, " porque no existe ]")
		}

	}

}

func generarIDMountYGuardar(path string, name string) {

	cadena := "vd"

	//verificar si existe
	existe := mapaMount[path]
	if existe != nil {

		//obtengo el indice, para saber que letra se le pone
		indice := 0
		var miArray []NodoMount
		i := -1
		for k, v := range mapaMount {
			i++
			if strings.Compare(strings.ToLower(k), strings.ToLower(path)) == 0 {
				indice = i  //obtengo indice
				miArray = v //obtengo array donde estan ids
			}
		}

		letra := indice

		if letra == 0 {
			cadena += "a"
		} else if letra == 1 {
			cadena += "b"
		} else if letra == 2 {
			cadena += "c"
		} else if letra == 3 {
			cadena += "d"
		} else if letra == 4 {
			cadena += "e"
		} else if letra == 5 {
			cadena += "f"
		} else if letra == 6 {
			cadena += "g"
		} else if letra == 7 {
			cadena += "h"
		} else if letra == 8 {
			cadena += "i"
		} else if letra == 9 {
			cadena += "j"
		} else if letra == 10 {
			cadena += "k"
		} else if letra == 11 {
			cadena += "l"
		} else if letra == 12 {
			cadena += "m"
		} else if letra == 13 {
			cadena += "n"
		} else if letra == 14 {
			cadena += "o"
		} else if letra == 15 {
			cadena += "p"
		} else if letra == 16 {
			cadena += "q"
		} else if letra == 17 {
			cadena += "r"
		} else if letra == 18 {
			cadena += "s"
		} else if letra == 19 {
			cadena += "t"
		} else if letra == 20 {
			cadena += "u"
		} else if letra == 21 {
			cadena += "v"
		} else if letra == 22 {
			cadena += "w"
		} else if letra == 23 {
			cadena += "x"
		} else if letra == 24 {
			cadena += "y"
		} else if letra == 25 {
			cadena += "z"
		}

		numParticion := len(miArray) + 1
		cadena += strconv.Itoa(numParticion)

		newMount := NodoMount{}
		newMount.Path = path
		newMount.Name = name
		newMount.PartID = cadena

		mapaMount[path] = append(mapaMount[path], newMount)

	} else {

		//se obtiene tamanio para generar la letra
		letra := len(mapaMount)

		if letra == 0 {
			cadena += "a"
		} else if letra == 1 {
			cadena += "b"
		} else if letra == 2 {
			cadena += "c"
		} else if letra == 3 {
			cadena += "d"
		} else if letra == 4 {
			cadena += "e"
		} else if letra == 5 {
			cadena += "f"
		} else if letra == 6 {
			cadena += "g"
		} else if letra == 7 {
			cadena += "h"
		} else if letra == 8 {
			cadena += "i"
		} else if letra == 9 {
			cadena += "j"
		} else if letra == 10 {
			cadena += "k"
		} else if letra == 11 {
			cadena += "l"
		} else if letra == 12 {
			cadena += "m"
		} else if letra == 13 {
			cadena += "n"
		} else if letra == 14 {
			cadena += "o"
		} else if letra == 15 {
			cadena += "p"
		} else if letra == 16 {
			cadena += "q"
		} else if letra == 17 {
			cadena += "r"
		} else if letra == 18 {
			cadena += "s"
		} else if letra == 19 {
			cadena += "t"
		} else if letra == 20 {
			cadena += "u"
		} else if letra == 21 {
			cadena += "v"
		} else if letra == 22 {
			cadena += "w"
		} else if letra == 23 {
			cadena += "x"
		} else if letra == 24 {
			cadena += "y"
		} else if letra == 25 {
			cadena += "z"
		}

		numParticion := len(mapaMount[path]) + 1
		cadena += strconv.Itoa(numParticion)

		newMount := NodoMount{}
		newMount.Path = path
		newMount.Name = name
		newMount.PartID = cadena

		mapaMount[path] = append(mapaMount[path], newMount)
	}

}

func mostrarMount() {

	fmt.Println("PARTICIONES MONTADAS:")

	for k, v := range mapaMount {
		//fmt.Println("key: ", k, " ---> ", v)
		array := v
		//recorro array de mount
		for i := 0; i < len(array); i++ {
			miMount := array[i]
			if strings.Compare(miMount.PartID, "eliminado") != 0 { //para que no muestra las particiones ya eliminadas
				fmt.Println("	id-> ", miMount.PartID, " path-> ", k, " name-> ", miMount.Name)
			}
		}
	}
}

func pruebaMount() {
	mapa := make(map[string][]NodoMount)

	uno := NodoMount{}
	uno.Path = "/home/"
	uno.Name = "Part1"
	uno.PartID = "vda1"

	dos := NodoMount{}
	dos.Path = "/home/gudiel"
	dos.Name = "Part2"
	dos.PartID = "vda2"

	tres := NodoMount{}
	tres.Path = "/home/gudiel"
	tres.Name = "Part3"
	tres.PartID = "vdb2"

	mapa["aja"] = append(mapa["aja"], uno)
	mapa["aja"] = append(mapa["aja"], dos)
	mapa["aja2"] = append(mapa["aja2"], tres)

	//imprimiendo IDs
	for i := 0; i < len(mapa["aja"]); i++ {
		fmt.Println(mapa["aja"][i])
	}

	//existe
	exist := mapa["aja"]
	if exist != nil {
		fmt.Println("existe")
	} else {
		fmt.Println("no existe")
	}

	for k := range mapa {
		fmt.Println("key: ", k, " ")
	}

	//tamanio
	fmt.Println(len(mapa["aja"]))
	fmt.Println(len(mapa["aja2"]))

	fmt.Println("ANTES DE ELIMINAR")
	for k, v := range mapa {
		fmt.Println("key: ", k, " ---> ", v)
	}

	part := "vda2"

	fmt.Println("\naaja")
	for k := range mapa { //recorre todas las path que hay en el mapa
		for i := 0; i < len(mapa[k]); i++ { //recorre el array mount, del path actual
			mountAcutal := mapa[k][i] //mount actual
			if strings.Compare(strings.ToLower(mountAcutal.PartID), strings.ToLower(part)) == 0 {
				//elimino
				vacioMount := NodoMount{}
				mapa[k][i] = vacioMount
			}
		}
	}

	fmt.Println("DESPUES DE ELIMINAR")
	for k, v := range mapa {
		fmt.Println("key: ", k, " ---> ", v)
	}

}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------INICIO UNMOUNT-------------------------------//
func unmountComando(index int) {
	var idn []string

	for i := index; i < len(listaComandos); i++ {

		if strings.Contains(strings.ToLower(listaComandos[i]), "id") {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				idn = append(idn, listaComandos[i+2]) //name
			} else {
				fmt.Print("\n[ ERROR: comando 'UNMOUNT' -> 'idn' ]")
			}
		}
	}

	desmontarParticion(idn)
}

func desmontarParticion(ids []string) {

	for x := 0; x < len(ids); x++ {
		bandera := false
		idEliminar := ids[x]

		for k := range mapaMount { //recorre todas las path que hay en el mapa
			for i := 0; i < len(mapaMount[k]); i++ { //recorre el array mount, del path actual
				mountAcutal := mapaMount[k][i] //mount actual
				if strings.Compare(strings.ToLower(mountAcutal.PartID), strings.ToLower(idEliminar)) == 0 {
					//elimino
					vacioMount := NodoMount{}
					vacioMount.PartID = "eliminado"
					mapaMount[k][i] = vacioMount
					fmt.Println("[ Se desmonto particion con nombre: ", idEliminar, " ]")
					bandera = true
				}
			}
		}

		if bandera == false {
			fmt.Print("\n[ ERROR: no se puede desmontar particion con nombre: ", idEliminar, " porque no existe ]")
		}
	}
}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------OPERACIONES PARA MBR-------------------------------//

//---------------------------------REPORTE MBR---------------------------------//

func reporteMBR(path string) {
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

	cadenaRep := "digraph { \n\n"
	cadenaRep += "tbl [ \n\n"
	cadenaRep += "shape=plaintext \n"
	cadenaRep += "label=< \n\n"
	cadenaRep += "<table color='orange' cellspacing='0'>\n"

	//cabecera de tabla
	cadenaRep += "<tr><td>NOMBRE</td><td>VALOR</td></tr>\n"

	//datos MBR
	fechaMbr := ""
	//si hay espacios en blanco en el [20]byte, osea espacios nulos, tira error
	for i := 0; i < 20; i++ {
		if m.Fecha[i] != 0 { //los que sean nulos no los concatena
			fechaMbr += string(m.Fecha[i])
		}
	}

	cadenaRep += "<tr><td>MBR_Tamanio</td><td>" + strconv.Itoa(int(m.Tamanio)) + "</td></tr>\n"
	cadenaRep += "<tr><td>MBR_Fecha</td><td>" + fechaMbr + "</td></tr>\n"
	cadenaRep += "<tr><td>MBR_Asignacion</td><td>" + strconv.Itoa(int(m.NumAsignacion)) + "</td></tr>\n"

	//recorro las estructuras primarias o extendidas
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		if actual.Tamanio != 0 {

			if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {

				//concateno atributos de particion extendida
				//eliminando espacios nulos del nombre
				nombrePart := ""
				for x := 0; x < 16; x++ {
					if actual.Name[x] != 0 { //los que sean nulos no los concatena
						nombrePart += string(actual.Name[x])
					}
				}

				cadenaRep += "<tr><td>" + nombrePart + "_Name</td><td>" + nombrePart + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Size</td><td>" + strconv.Itoa(int(actual.Tamanio)) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Tipe</td><td>" + string(actual.TipoParticion) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Fit</td><td>" + string(actual.TipoAjuste[:]) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Start</td><td>" + strconv.Itoa(int(actual.Start)) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Status</td><td>" + strconv.Itoa(int(actual.Estado)) + "</td></tr>\n"

				//para las logicas
				misParticionesLogicas := actual.ParticionesLogicas

				//recorro las particiones logicas
				for x := 0; x < len(misParticionesLogicas); x++ {
					temp := misParticionesLogicas[x]

					if temp.Tamanio != 0 {

						//elimino espacios nulos del nombre
						nombrePartLog := ""
						for y := 0; y < 16; y++ {
							if temp.Name[y] != 0 { //los que sean nulos no los concatena
								nombrePartLog += string(temp.Name[y])
							}
						}

						cadenaRep += "<tr><td>" + nombrePartLog + "_Name</td><td>" + nombrePartLog + "</td></tr>\n"
						cadenaRep += "<tr><td>" + nombrePartLog + "_Size</td><td>" + strconv.Itoa(int(temp.Tamanio)) + "</td></tr>\n"
						cadenaRep += "<tr><td>" + nombrePartLog + "_Tipe</td><td>" + string(temp.TipoParticion) + "</td></tr>\n"
						cadenaRep += "<tr><td>" + nombrePartLog + "_Fit</td><td>" + string(temp.TipoAjuste[:]) + "</td></tr>\n"
						cadenaRep += "<tr><td>" + nombrePartLog + "_Start</td><td>" + strconv.Itoa(int(temp.Start)) + "</td></tr>\n"
						cadenaRep += "<tr><td>" + nombrePartLog + "_Status</td><td>" + strconv.Itoa(int(temp.Estado)) + "</td></tr>\n"

					}
				}

				//para las primarias
			} else {

				//eliminando espacios nulos
				nombrePart := ""
				for x := 0; x < 16; x++ {
					if actual.Name[x] != 0 { //los que sean nulos no los concatena
						nombrePart += string(actual.Name[x])
					}
				}

				cadenaRep += "<tr><td>" + nombrePart + "_Name</td><td>" + nombrePart + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Size</td><td>" + strconv.Itoa(int(actual.Tamanio)) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Tipe</td><td>" + string(actual.TipoParticion) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Fit</td><td>" + string(actual.TipoAjuste[:]) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Start</td><td>" + strconv.Itoa(int(actual.Start)) + "</td></tr>\n"
				cadenaRep += "<tr><td>" + nombrePart + "_Status</td><td>" + strconv.Itoa(int(actual.Estado)) + "</td></tr>\n"

			}
		}
	}

	cadenaRep += "</table>\n\n"
	cadenaRep += ">];\n"
	cadenaRep += "}"

	crearDot("report_mbr", cadenaRep)
	crearImg("report_mbr")
}

func reporteDisk(path string) {
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

	cadenaRep := "digraph { \n\n"
	cadenaRep += "tbl [ \n\n"
	cadenaRep += "shape=plaintext \n"
	cadenaRep += "label=< \n\n"
	cadenaRep += "<table border='0' cellborder='1' color='blue' cellspacing='0'>\n\n"

	//para las primarias y extendida
	//que dividiria asi -> |MBR|PRIMARIA|PRIMAR|EXTEND|LIBRE|PRIMAR|
	cadenaRep += "<tr>\n\n"

	//MBR
	cadenaRep += "<td rowspan='2' bgcolor='yellow'>MBR</td>"

	//verificando si hay particiones
	hayParticion := false
	for i := 0; i < len(misParticiones); i++ {
		actual := misParticiones[i]
		if actual.Tamanio != 0 {
			hayParticion = true
			break
		}
	}

	//disco vacio
	if hayParticion == false {

		//concatenamos vacio despues del MBR
		cadenaRep += "<td rowspan='2' bgcolor='yellow'>MBR</td><td rowspan='2' bgcolor='green'>LIBRE</td> </tr>"

		//hay particion
	} else {

		for i := 0; i < len(misParticiones); i++ {

			actual := misParticiones[i]

			if actual.Tamanio != 0 {

				//si es la posicion 0
				if i == 0 {
					tamMBR := size
					starActual := misParticiones[i].Start
					espacio := starActual - int64(tamMBR)
					//si hay espacio libre
					if espacio > 1 {
						//si es primaria
						if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
							//concateno la libre y la primaria
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

						} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
							//concateno la libre y la primaria
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
						}
					} else {
						//si es primaria
						if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
							//concateno primaria
							cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
						} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
							//concateno la libre y la primaria
							cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
						}
					}

					//si es la ultima posicion
				} else if i == len(misParticiones)-1 {

					posAnterior := -1
					for x := i - 1; x > -1; x-- { //empieza a buscar una antes
						//si encuantra antes una particion, guardo posicion donde la encuentra
						if misParticiones[x].Tamanio != 0 {
							posAnterior = x
							break
						}
					}

					//si hay una particion antes
					if posAnterior != -1 {
						tamDisco := m.Tamanio
						tamMBR := int64(size)
						starAnterior := misParticiones[posAnterior].Start
						tamAnterior := misParticiones[posAnterior].Tamanio
						starActual := misParticiones[i].Start
						tamActual := misParticiones[i].Tamanio

						//espacio entre el anterior y el ultimo
						espacio := starActual - (starAnterior + tamAnterior)
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}

						//espacio entre el ultimo y el disco
						espacio2 := (tamDisco - tamMBR) - (starActual + tamActual)
						if espacio2 > 1 {
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td>"
						}

						//si no hay una particion antes
					} else if posAnterior == -1 {
						tamDisco := m.Tamanio
						tamMBR := int64(size)
						starActual := misParticiones[i].Start
						tamActual := misParticiones[i].Tamanio

						espacio := starActual - tamMBR
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}

						//espacio entre el ultimo y el disco
						espacio2 := (tamDisco - tamMBR) - (starActual + tamActual)
						if espacio2 > 1 {
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td>"
						}

					}

					//no es primera ni ultima
				} else {

					posAnterior := -1
					for x := i - 1; x > -1; x-- { //empieza a buscar una antes
						//si encuantra antes una particion, guardo posicion donde la encuentra
						if misParticiones[x].Tamanio != 0 {
							posAnterior = x
							break
						}
					}

					posSiguiente := -1
					for x := i + 1; x < len(misParticiones); x++ { //empieza a buscar una despues, de las dos donde se encontro espacio
						//si encuantra un espacio
						if misParticiones[x].Tamanio != 0 {
							posSiguiente = x
							break
						}
					}

					//tiene anterior pero no siguiente
					if posAnterior != -1 && posSiguiente == -1 {
						fmt.Println("ujuaa")
						tamDisco := m.Tamanio
						tamMBR := int64(size)
						starAnterior := misParticiones[posAnterior].Start
						tamAnterior := misParticiones[posAnterior].Tamanio
						starActual := misParticiones[i].Start
						tamActual := misParticiones[i].Tamanio

						//espacio entre ambas
						espacio := starActual - (starAnterior + tamAnterior)
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}

						//como no tiene siguiente
						//espacio entre el ultimo y el disco
						espacio2 := (tamDisco - tamMBR) - (starActual + tamActual)

						if espacio2 > 1 {
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td>"
						}

						//no tiene anterior, pero si siguiente
					} else if posAnterior == -1 && posSiguiente != -1 {

						tamMBR := int64(size)
						starActual := misParticiones[i].Start

						//espacio entre MBR y actual
						espacio := starActual - tamMBR
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}

						//no tiene anterior, ni siguiente
					} else if posAnterior == -1 && posSiguiente == -1 {
						tamDisco := m.Tamanio
						tamMBR := int64(size)
						starActual := misParticiones[i].Start
						tamActual := misParticiones[i].Tamanio

						//espacio entre MBR y actual
						espacio := starActual - tamMBR
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}

						//como no tiene siguiente
						//espacio entre el ultimo y el disco
						espacio2 := (tamDisco - tamMBR) - (starActual + tamActual)
						if espacio2 > 1 {
							cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td>"
						}

						//tiene anterior y siguiente
					} else if posAnterior != -1 && posSiguiente != -1 {
						starAnterior := misParticiones[posAnterior].Start
						tamAnterior := misParticiones[posAnterior].Tamanio
						starActual := misParticiones[i].Start

						espacio := starActual - (starAnterior + tamAnterior)
						if espacio > 1 {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='pink' rowspan='2'>PRIMARIA</td>"

							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td rowspan='2' bgcolor='green'>LIBRE</td><td bgcolor='cyan'>EXTENDIDA</td>"
							}
						} else {
							//si es primaria
							if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "p") == 0 {
								//concateno primaria
								cadenaRep += "<td bgcolor='pink' rowspan='2'>PRIMARIA</td>"
							} else if strings.Compare(strings.ToLower(string(misParticiones[i].TipoParticion)), "e") == 0 {
								//concateno la libre y la primaria
								cadenaRep += "<td bgcolor='cyan'>EXTENDIDA</td>"
							}
						}
					}
				}
			}
		}
	}

	cadenaRep += "</tr>\n\n"

	//EMPIEZA LOGICAS

	cadenaRep += "<tr><td cellpadding='1'>\n\n"
	cadenaRep += "<table bgcolor='orange' cellspacing='0'>\n\n"
	cadenaRep += "<tr>\n"

	posicionExtendida := -1
	for i := 0; i < len(misParticiones); i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//si existe una extendida
	if posicionExtendida != -1 {
		hayLogicas := false
		misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas
		for i := 0; i < len(misParticionesLogicas); i++ {
			if misParticionesLogicas[i].Tamanio != 0 {
				hayLogicas = true
				break
			}
		}

		if hayLogicas == false {
			cadenaRep += "<td bgcolor='green'>LIBRE</td>\n"
		}
	}

	cadenaRep += "</tr>"
	cadenaRep += "</table>\n"
	cadenaRep += "</td></tr>\n\n"

	cadenaRep += "</table>\n\n"
	cadenaRep += ">];\n\n"
	cadenaRep += "}"

	crearDot("report_Disk", cadenaRep)
	crearImg("report_Disk")

}

func crearDot(name string, cadena string) {
	f := createFile(name + ".dot")
	defer closeFile(f)
	writeFile(f, cadena)
}

func createFile(p string) *os.File {
	fmt.Println("crear")
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	return f
}

func writeFile(f *os.File, cadena string) {
	fmt.Println("escribir")
	fmt.Fprintln(f, cadena)

}

func closeFile(f *os.File) {
	fmt.Println("cerrar")
	f.Close()
}

func crearImg(name string) {
	//ejecutando comando, en tiempo de ejecucion
	cmd := exec.Command("dot", "-Tpng", name+".dot", "-o", name+".png")
	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		}
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		//fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
	}
}

//-------------------------------FIN REPORTE MBR-------------------------------//

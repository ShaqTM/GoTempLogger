package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

const DB_CONNECT_STRING = "host=localhost port=5432 user=postgres password=Mm000000 dbname=logger sslmode=disable"
const DB_CONNECT_STRING_INIT = "host=localhost port=5432 user=postgres password=Mm000000 dbname=postgres sslmode=disable"
const DEVICE_NAME = "esp8266"

func main() {
	init_database()
	db, err := sql.Open("postgres", DB_CONNECT_STRING)

	if err != nil {
		fmt.Printf("Database opening error -->%v\n", err)
		panic("Database error")
	}
	defer db.Close()

	init_table(&db)
	go readFromDevice(DEVICE_NAME, &db)

	http.Handle("/", handleRoot(&db))
	http.Handle("/getLastData", handlegetLastData(&db))
	http.Handle("/chart", handleChart(&db))
	http.Handle("/getDataArray", handlegetDataArray(&db))

	http.ListenAndServe(":5000", nil)
	for {

	}
}

func findDevice(devName string) string {
	requestArray := buildRequest(devName)
	ifaces := externalIP()
	for _, iface := range ifaces {
		devIP := sendMultiCast(iface, requestArray, devName)
		if devIP != "" {
			return devIP
		}
	}
	return ""

}

func sendMultiCast(iface net.Interface, requestArray []byte, devName string) string {
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.251:5353")
	if err != nil {
		fmt.Printf("Address not resolved!", err.Error())
		return ""
	}
	i_addr := getIP(iface)
	_, err = net.ResolveUDPAddr("udp", i_addr+":5353")
	if err != nil {
		fmt.Printf("Local address not resolved!", err.Error())
		return ""
	}
	conn, err := net.ListenMulticastUDP("udp", &iface, addr)
	conn.SetReadBuffer(8000)
	if err != nil {
		fmt.Println("Listen multicast. Dial not sucsesfull!", err.Error())
		return ""
	}
	defer conn.Close()

	timeout := true
	for i := 0; i < 10; i++ {
		if timeout {
			fmt.Println("Sending mDNS request from IP: ", i_addr)
			_, err := conn.WriteToUDP(requestArray, addr)
			if err != nil {
				fmt.Println("WriteToUDP not sucsesfull!", err.Error())
				return ""
			}
		}
		timeout = false
		buffer := make([]byte, 8000)

		err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			fmt.Println("SetReadDeadLine not sucsesfull!", err.Error())
			return ""
		}
		_, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			e, _ := err.(net.Error)
			if e.Timeout() {
				timeout = true
				continue
			}
			fmt.Println("ReadFromUDP failed:", err.Error())
			return ""
		}

		devIP := parseAnswer(buffer, devName)
		if devIP != "" {
			fmt.Println("Found device with IP = ", devIP)
			return devIP
		}

	}
	return ""

}

func buildRequest(devName string) []byte {
	var requestArray []byte
	//ID
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//Flags
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of questions
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	//number of answers
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of ///
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of ///
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//device name
	//writeName("esp8266._http._tcp.local", message)
	requestArray = addStringToArray(devName, requestArray)
	requestArray = addStringToArray("_http", requestArray)
	requestArray = addStringToArray("_tcp", requestArray)
	requestArray = addStringToArray("local", requestArray)
	requestArray = append(requestArray, 0)

	//	//type
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 33)
	//	//class
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	return requestArray

}

func addStringToArray(str string, requestArray []byte) []byte {
	tmpArray := []byte(str)
	lenStr := byte(len(tmpArray))
	requestArray = append(requestArray, lenStr)
	requestArray = append(requestArray, tmpArray...)
	return requestArray
}

func parseAnswer(buffer []byte, devName string) string {
	const base int = 256
	var reqNum int = base*(int)(buffer[4]) + (int)(buffer[5])
	var ansNum int = base*(int)(buffer[6]) + (int)(buffer[7])
	var str string
	var blockBegin int = 12
	var i int
	var ansType int
	var resLen int
	var ip string
	fmt.Println("Parse answer")
	fmt.Println("ansNum: ", ansNum)
	fmt.Println("reqNum: ", reqNum)

	for i = 0; i < reqNum; i++ {
		str, blockBegin = readString(blockBegin, buffer)
		fmt.Println("reg: ", str)
		blockBegin += 4

	}
	if ansNum != 0 {
		str, blockBegin = readString(blockBegin, buffer)
		fmt.Println("Answer string: ", str)
		if str != devName+"._http._tcp.local" {
			return ""
		}
	}
	for i = 0; i < ansNum; i++ {
		fmt.Println("ansType,blockBegin: ", blockBegin)
		fmt.Println("ansType: ", buffer[blockBegin], ",", buffer[blockBegin+1])
		ansType = base*(int)(buffer[blockBegin]) + (int)(buffer[blockBegin+1])
		blockBegin += 2
		blockBegin += 6
		resLen = base*(int)(buffer[blockBegin]) + (int)(buffer[blockBegin+1])
		fmt.Println("resLen: ", resLen)
		blockBegin += 2
		if ansType == 33 {
			blockBegin += 6
			str, blockBegin = readString(blockBegin, buffer)
			fmt.Println("ansType=33: ", str)
			str, blockBegin = readString(blockBegin, buffer)
			fmt.Println("ansType=33: ", str)

		} else if ansType == 1 {
			fmt.Println("IP: ", buffer[blockBegin], ",", buffer[blockBegin+1], ",", buffer[blockBegin+2], ",", buffer[blockBegin+3])
			ip = strconv.Itoa((int)(buffer[blockBegin])) + "." + strconv.Itoa((int)(buffer[blockBegin+1])) + "." + strconv.Itoa((int)(buffer[blockBegin+2])) + "." + strconv.Itoa((int)(buffer[blockBegin+3]))
			fmt.Println("IP: ", ip)
			return ip

		} else {
			blockBegin += resLen
		}

	}
	return ""
}

func readString(reqBegin int, buffer []byte) (string, int) {
	reqStr := ""
	var strLen int
	var i int
	position := reqBegin
	for buffer[position] != 0 {
		if reqStr != "" {
			reqStr += "."
		}
		strLen = (int)(buffer[position])
		for i = 1; i <= strLen; i++ {
			reqStr += string(buffer[position+i])
		}
		position += strLen
		position += 1
	}
	position += 1
	return reqStr, position
}

func externalIP() []net.Interface {
	var ifArray []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return ifArray
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		_, err := iface.Addrs()
		if err != nil {
			continue
		}
		ifArray = append(ifArray, iface)
	}
	return ifArray
}

func getIP(iface net.Interface) string {
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue // not an ipv4 address
		}
		return ip.String()
		fmt.Println("Local IP:", ip.String())
	}
	return ""

}

func init_database() {
	db, err := sql.Open("postgres", DB_CONNECT_STRING_INIT)

	if err != nil {
		fmt.Println("Database opening error -->%v\n", err)
		panic("Database error")
	}
	defer db.Close()

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false AND datname = 'logger';")

	if err != nil {
		fmt.Println("Error serching database:", err)
		panic("Error serching database")
	}

	for rows.Next() {
		fmt.Println("Database logger found")
		return
	}
	_, err = db.Exec("CREATE DATABASE logger WITH OWNER postgres;")
	if err != nil {
		fmt.Println("Error creating database:", err)
		panic("Error creating database")
	}
	fmt.Println("Database created successfully")
}

func init_table(pdb **sql.DB) {
	db := *pdb

	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'logger'")

	if err != nil {
		fmt.Println("Error serching table:", err)
		panic("Error serching database")
	}

	for rows.Next() {
		fmt.Println("Table logger found")
		return
	}

	init_table_string := `CREATE TABLE IF NOT EXISTS public.log_data(
         id serial,
         device_name varchar(10) not null,
         parameter_name varchar(20) not null,
         value numeric(6,2),
         event_time_id integer,
         constraint id_data primary key (id));
		CREATE TABLE IF NOT EXISTS public.log_time(
         id serial,
         event_time timestamp default current_timestamp,
         constraint id_time primary key (id));
		
		`

	_, err = db.Exec(init_table_string)
	if err != nil {
		fmt.Println("Table create error", err)
		panic("Table create error")
	}

	fmt.Println("Table created successfully")

}

func insert_data(pdb **sql.DB, response []byte, device_name string) {
	const INSERT_DATA_QUERY = `insert into public.log_data(device_name, parameter_name, value,event_time_id)
                                  values ($1, $2, $3, $4);`
	const INSERT_TIME_QUERY = `insert into public.log_time DEFAULT VALUES RETURNING id;`

	var message interface{}
	db := *pdb

	err := json.Unmarshal(response, &message)
	if err != nil {
		fmt.Println("Error decoding json: ", err)
		return
	}
	LastInsertId := 0
	err = db.QueryRow(INSERT_TIME_QUERY).Scan(&LastInsertId)
	if err != nil {
		fmt.Println("Error inserting time data: ", err)
		return
	}
	m := message.(map[string]interface{})
	for key, value := range m {
		_, err = db.Exec(INSERT_DATA_QUERY, device_name, key, value, LastInsertId)
		if err != nil {
			fmt.Println("Error inserting data: ", err)
		}
	}

}

func get_devices(pdb **sql.DB) []string {
	var device_list []string
	db := *pdb

	rows, err := db.Query("SELECT DISTINCT device_name FROM log_data")
	if err != nil {
		fmt.Println("Error query device names: ", err)
		return device_list
	}
	dev_name := ""
	for rows.Next() {
		err = rows.Scan(&dev_name)
		if err != nil {
			fmt.Println("Error getting device name: ", err)
			continue
		}
		device_list = append(device_list, dev_name)
	}
	return device_list

}

func get_last_data(pdb **sql.DB, device_name string, datetime string) string {
	db := *pdb
	id := 0
	whereText := ""
	if datetime != "" {
		whereText = fmt.Sprintf("WHERE log_time.event_time<='%s'", datetime)
	}
	queryText := fmt.Sprintf(`SELECT log_time.id 
	FROM log_time 
	INNER JOIN log_data ON log_data.event_time_id = log_time.id
	AND log_data.device_name='%s'
	%s
	ORDER BY log_time.id DESC
	LIMIT 1`, device_name, whereText)
	fmt.Println(queryText)
	err := db.QueryRow(queryText).Scan(&id)
	if err != nil {
		fmt.Println("Error query last data: ", err)
		return ""
	}
	queryText = fmt.Sprintf("SELECT parameter_name,value FROM log_data WHERE event_time_id=%d ORDER BY parameter_name", id)
	fmt.Println(queryText)
	rows, err := db.Query(queryText)
	if err != nil {
		fmt.Println("Error query last data: ", err)
		return ""
	}
	const labelString = `<p><label>%s: %f</label></p>`
	var parameter_value float32
	parameter_name := ""
	returnString := ""
	for rows.Next() {

		err = rows.Scan(&parameter_name, &parameter_value)
		if err != nil {
			fmt.Println("Error query last data: ", err)
			continue
		}
		fmt.Println(parameter_name)
		fmt.Println(parameter_value)
		returnString = returnString + fmt.Sprintf(labelString, parameter_name, parameter_value)
	}
	return returnString

}

type RespNode struct {
	Time string
	Data []float32
}
type RespStruct struct {
	Parameters       []string
	ParametersNumber int
	Data             []RespNode
}

func get_data_array(pdb **sql.DB, device_name string, datetime1 string, datetime2 string) string {
	db := *pdb

	whereText := "WHERE True "
	if datetime1 != "" {
		whereText = whereText + fmt.Sprintf(" AND log_time.event_time>='%s'", datetime1)
	}
	if datetime2 != "" {
		whereText = whereText + fmt.Sprintf(" AND log_time.event_time<='%s'", datetime2)
	}

	fmt.Println(whereText)
	queryText := fmt.Sprintf(`SELECT DISTINCT
		log_data.parameter_name
	FROM log_time 
	INNER JOIN log_data ON log_data.event_time_id = log_time.id
	AND log_data.device_name='%s'
	%s
	ORDER BY log_data.parameter_name`, device_name, whereText)
	rows, err := db.Query(queryText)
	if err != nil {
		fmt.Println("Error query parameter list: ", err)
		return ""
	}
	parameter_name := ""
	parameters := make(map[string]int)
	var parametersArray []string
	paramsNumber := 0
	for rows.Next() {
		err = rows.Scan(&parameter_name)
		if err != nil {
			fmt.Println("Error query parameter name: ", err)
			return ""
		}
		parameters[parameter_name] = paramsNumber
		paramsNumber++
		fmt.Println(parameter_name)
		parametersArray = append(parametersArray, parameter_name)
	}

	queryText = fmt.Sprintf(`SELECT log_time.event_time,
		log_data.parameter_name,
		log_data.value
	FROM log_time 
	INNER JOIN log_data ON log_data.event_time_id = log_time.id
	AND log_data.device_name='%s'
	%s
	ORDER BY log_time.id ASC`, device_name, whereText)
	fmt.Println(queryText)
	rows, err = db.Query(queryText)
	if err != nil {
		fmt.Println("Error query data array: ", err)
		return ""
	}
	var parameter_value float32
	parameter_name = ""
	prev_event_time := ""
	event_time := ""

	data := make([]float32, paramsNumber)
	for i := 0; i < paramsNumber; i++ {
		data[i] = 0
	}

	var nodeArray []RespNode
	for rows.Next() {
		err = rows.Scan(&event_time, &parameter_name, &parameter_value)
		if err != nil {
			fmt.Println("Error query last data: ", err)
			continue
		}
		if event_time != prev_event_time {
			node := RespNode{Data: data, Time: prev_event_time}
			nodeArray = append(nodeArray, node)
			data = make([]float32, paramsNumber)
			for i := 0; i < paramsNumber; i++ {
				data[i] = 0
			}

		}
		prev_event_time = event_time
		data[parameters[parameter_name]] = parameter_value
	}
	node := RespNode{Data: data, Time: prev_event_time}
	nodeArray = append(nodeArray, node)
	//fmt.Println("%s; %f; %f; %f; %f", node.time, node.data[0], node.data[1], node.data[2], node.data[3])
	respStruct := &RespStruct{Parameters: parametersArray, ParametersNumber: paramsNumber, Data: nodeArray}
	resJSON, err := json.Marshal(respStruct)
	if err != nil {
		fmt.Println("Error query last data: ", err)
		return "Error query last data: " + err.Error()
	}
	fmt.Println(string(resJSON))
	return string(resJSON)

}

func readFromDevice(device_name string, pdb **sql.DB) {
	devIP := ""
	for {
		for devIP == "" {
			devIP = findDevice(device_name)
			if devIP == "" {
				time.Sleep(time.Second * 10)
			}
		}

		timeout := time.Duration(10 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get("http://" + devIP + "/tempData")
		if err != nil {
			devIP := findDevice(device_name)
			if devIP != "" {
				resp, err = client.Get("http://" + devIP + "/tempData")
			}
		}
		if err == nil {
			responseText, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				fmt.Println("Response: ", string(responseText))
				insert_data(pdb, responseText, device_name)
			}
		}
		time.Sleep(time.Second * 5)
	}
}

const rootHTML = `
<!DOCTYPE HTML>
<html>
  <head>
    <meta charset="utf-8">
    <title>Simple Go Web App</title>
  </head>
  <body>
	<p>
		<label for="device_list">Выберите устройство:</label>
		<select id="device_list" name="device_list">
			<option value=""></option>
			%s
		</select>
	</p>
	<p>
		<input type="button" id="refresh" value="Обновить"/>
		</p>
	<p>
		<label for="date">Введите дату и время:</label>
		<input type="date" id="date">
		<input type="time" id="time">
		<script>
		
		var refreshData = function() {
			var data_block = document.getElementById("data_block")
		    var request = new XMLHttpRequest();
			var datetime = "";
			if (date.value!=""){
				datetime = date.value+"T";
				if (time.value!=""){
					datetime = datetime+time.value;
				}
				else{
					datetime = datetime+"00:00:00";
				}
			} 
    		request.open('GET','getLastData?device='+device_list.options[device_list.selectedIndex].value+'&datetime='+datetime,true);
    		request.addEventListener('readystatechange', function() {
      			if ((request.readyState==4) && (request.status==200)) {
        			data_block.innerHTML = request.responseText;
      			}
    		}); 
			request.send();			
		};
		var device_list = document.getElementById("device_list")
		device_list.onchange = refreshData		
		var refresh = document.getElementById("refresh")
		refresh.onclick = refreshData		
		var date = document.getElementById("date")
		var time = document.getElementById("time")
		date.addEventListener("input",refreshData)
		time.addEventListener("input",refreshData)
		
		</script>		
	</p>
	<p>
		<div id="data_block"
		</div>
	</p>
		<form action="chart">
    		<input type="submit" value="Show chart" />
		</form>		
	</body>
</html>`

const chartHTML = `
<!DOCTYPE HTML>
<html>
  <head>
    <meta charset="utf-8">
    <title>Simple Go Web App</title>
	<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
    <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      function drawChart(resp) {
		var data = new google.visualization.DataTable();
		data.addColumn('datetime', 'Time');
		for (var i=0; i<resp.paramsNumber;i++){
			data.addColumn('number', resp.Parameters[i]);
		}
		for (var counter = 0;counter<=data.Data.length;counter++){
			data.addRows();
			data.setCell(counter,0,resp.Data[counter].Time);
			for (var i=0; i<resp.paramsNumber;i++){
				data.setCell(counter,i+0,resp.Data[counter].Data[i]);
			}
			
		}
        var options = {
          title: 'Company Performance',
          curveType: 'function',
          legend: { position: 'bottom' }
        };

        var chart = new google.visualization.LineChart(document.getElementById('curve_chart'));

        chart.draw(data, options);
      }
    </script>
	
  </head>
  <body>
	<p>
		<label for="device_list">Выберите устройство:</label>
		<select id="device_list" name="device_list">
			<option value=""></option>
			%s
		</select>
	</p>
	<p>
		<input type="button" id="refresh" value="Обновить"/>
		</p>
	<p>
		<label for="datetime1">Период с:</label>
		<input type="date" id="date1">
		<input type="time" id="time1">
		<label for="datetime2">по:</label>
		<input type="date" id="date2">
		<input type="time" id="time2">
		<script>
		
		var refreshData = function() {
			var data_block = document.getElementById("data_block")
		    var request = new XMLHttpRequest();
			var datetime1 = "";
			if (date1.value!=""){
				datetime1 = date1.value+"T";
				if (time1.value!=""){
					datetime1 = datetime1+time1.value;
				}
				else{
					datetime1 = datetime1+"00:00:00";
				}
			} 
			var datetime2 = "";
			if (date2.value!=""){
				datetime2 = date2.value+"T";
				if (time2.value!=""){
					datetime2 = datetime2+time2.value;
				}
				else{
					datetime2 = datetime2+"00:00:00";
				}
			} 
    		request.open('GET','getDataArray?device='+device_list.options[device_list.selectedIndex].value+'&datetime1='+datetime1+'&datetime2='+datetime2,true);
    		request.addEventListener('readystatechange', function() {
      			if ((request.readyState==4) && (request.status==200)) {
        			drawChart(JSON.parse(request.responseText));
      			}
    		}); 
			request.send();			
		};
		var device_list = document.getElementById("device_list")
		//device_list.onchange = refreshData		
		var refresh = document.getElementById("refresh")
		refresh.onclick = refreshData		
		var date1 = document.getElementById("date1")
		var time1 = document.getElementById("time1")
		var date2 = document.getElementById("date2")
		var time2 = document.getElementById("time2")
		</script>		
	</p>
	<p>
		<div id="curve_chart" style="width: 900px; height: 500px"></div>
	</p>

	</body>
</html>`

const optionHTML = `
<option value="%s">%s</option>`

func handleRoot(pdb **sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options := ""
		device_list := get_devices(pdb)
		for _, device_name := range device_list {
			options = options + fmt.Sprintf(optionHTML, device_name, device_name)
		}
		fmt.Println(options)
		fmt.Fprintf(w, rootHTML, options)
	})
}

func handlegetLastData(pdb **sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		device_name := r.URL.Query().Get("device")
		datetime := r.URL.Query().Get("datetime")
		fmt.Println(datetime)
		data := get_last_data(pdb, device_name, datetime)
		fmt.Println(data)
		fmt.Fprintf(w, data)
	})
}

func handlegetDataArray(pdb **sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		device_name := r.URL.Query().Get("device")
		datetime1 := r.URL.Query().Get("datetime1")
		datetime2 := r.URL.Query().Get("datetime2")
		fmt.Println(datetime1)
		data := get_data_array(pdb, device_name, datetime1, datetime2)

		fmt.Fprintf(w, data)
	})
}
func handleChart(pdb **sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options := ""
		device_list := get_devices(pdb)
		for _, device_name := range device_list {
			options = options + fmt.Sprintf(optionHTML, device_name, device_name)
		}
		fmt.Println(options)
		fmt.Fprintf(w, chartHTML, options)
	})
}

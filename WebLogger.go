package main

import (
	"Weblogger/device"
	"Weblogger/mdatabase"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

const DB_CONNECT_STRING = "host=localhost port=5432 user=postgres password=Mm000000 dbname=logger sslmode=disable"
const DB_CONNECT_STRING_INIT = "host=localhost port=5432 user=postgres password=Mm000000 dbname=postgres sslmode=disable"

type Configuration struct {
	Devices []device.Device
}

func main() {
	init_database()
	db, err := sql.Open("postgres", DB_CONNECT_STRING)

	if err != nil {
		fmt.Printf("Database opening error -->%v\n", err)
		panic("Database error")
	}
	defer db.Close()
	mdb := mdatabase.MDB{Pdb: &db}
	mdb.Init_table()

	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	for _, device := range configuration.Devices {
		go device.ReadFromDevice(mdb)
	}

	http.Handle("/", handleRoot(mdb))
	http.Handle("/getLastData", handlegetLastData(mdb))
	http.Handle("/chart", handleChart(mdb))
	http.Handle("/getDataArray", handlegetDataArray(mdb))

	http.ListenAndServe(":5000", nil)
	for {

	}
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
      google.charts.load('current', {'packages':['line']});
      function drawChart(resp) {
		var data = new google.visualization.DataTable();
		data.addColumn('datetime', 'Time');
		for (var i=0; i<resp.ParametersNumber;i++){
			data.addColumn('number', resp.Parameters[i]);
		}
		for (var counter = 0;counter<resp.Data.length;counter++){
			data.addRow();
			data.setCell(counter,0,new Date(resp.Data[counter].Time));
			for (var i=0; i<resp.ParametersNumber;i++){
				data.setCell(counter,i+1,resp.Data[counter].Data[i]);
			}
			
		}
        var options = {
          title: 'Log',
        hAxis: {
          gridlines: {
            count: -1,
            units: {
              days: {format: ['MMM dd']},
              hours: {format: ['HH:mm', 'ha']},
            }
          },
          minorGridlines: {
            units: {
              hours: {format: ['hh:mm:ss a', 'ha']},
              minutes: {format: ['HH:mm a Z', ':mm']}
            }
          }
        },
  
          legend: { position: 'bottom' }
        };

        var chart = new google.charts.Line(document.getElementById('curve_chart'));
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

func handleRoot(mdb mdatabase.MDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options := ""
		device_list := mdb.Get_devices()
		for _, device_name := range device_list {
			options = options + fmt.Sprintf(optionHTML, device_name, device_name)
		}
		fmt.Println(options)
		fmt.Fprintf(w, rootHTML, options)
	})
}

func handlegetLastData(mdb mdatabase.MDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		device_name := r.URL.Query().Get("device")
		datetime := r.URL.Query().Get("datetime")
		fmt.Println(datetime)
		data := mdb.Get_last_data(device_name, datetime)
		fmt.Println(data)
		fmt.Fprintf(w, data)
	})
}

func handlegetDataArray(mdb mdatabase.MDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		device_name := r.URL.Query().Get("device")
		datetime1 := r.URL.Query().Get("datetime1")
		datetime2 := r.URL.Query().Get("datetime2")
		fmt.Println(datetime1)
		data := mdb.Get_data_array(device_name, datetime1, datetime2)

		fmt.Fprintf(w, data)
	})
}
func handleChart(mdb mdatabase.MDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options := ""
		device_list := mdb.Get_devices()
		for _, device_name := range device_list {
			options = options + fmt.Sprintf(optionHTML, device_name, device_name)
		}
		fmt.Println(options)
		fmt.Fprintf(w, chartHTML, options)
	})
}

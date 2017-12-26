package mdatabase

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
)

type MDB struct {
	Pdb **sql.DB
}

func (mdb MDB) Init_table() {
	db := *mdb.Pdb

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

func (mdb MDB) Insert_data(response []byte, device_name string) {
	const INSERT_DATA_QUERY = `insert into public.log_data(device_name, parameter_name, value,event_time_id)
                                  values ($1, $2, $3, $4);`
	const INSERT_TIME_QUERY = `insert into public.log_time DEFAULT VALUES RETURNING id;`

	var message interface{}
	db := *mdb.Pdb

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

func (mdb MDB) Get_devices() []string {
	var device_list []string
	db := *mdb.Pdb

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

func (mdb MDB) Get_last_data(device_name string, datetime string) string {
	db := *mdb.Pdb
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

func (mdb MDB) Get_data_array(device_name string, datetime1 string, datetime2 string) string {
	db := *mdb.Pdb

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
		event_time = event_time[:19]
		if event_time != prev_event_time && prev_event_time != "" {
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

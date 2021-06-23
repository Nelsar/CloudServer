package influx

import (
	"fmt"
	"gitlab.citicom.kz/CloudServer/server/models"
	"math"
	"reflect"
	"strconv"
	"time"
)

func (influx *Influx) SaveSensorData(sensorTagName string, value interface{}, createdTs int64) error {
	tags := map[string]string{
		"tagName": sensorTagName,
	}
	createdTime := time.Unix(createdTs, 0)

	return influx.Exec(
		tags,
		map[string]interface{}{
			"value": value,
		},
		createdTime,
	)
}

type singleGraphData struct {
	XColumn []int64   `json:"xColumn"`
	YColumn []float64 `json:"yColumn"`
}

type ResultGraphData struct {
	Columns [][]interface{}        `json:"columns"`
	Objects []*models.SensorResult `json:"objects"`
}

//groupTime values = 1d, 1h, 1m, 1s
//selectTime values = 1d, 1h, 1m, 1s | result > now() - select time
func (influx *Influx) GetSensorMedianData(sensorTagName string, selectTime string, diffTime string, groupTime string) (*singleGraphData, error) {
	query := fmt.Sprintf(`SELECT MEDIAN(value) FROM cloudData WHERE tagName='%v' AND time >= now() - %v GROUP BY time(%v)`, sensorTagName, selectTime, groupTime)

	if len(diffTime) > 0 {
		query = fmt.Sprintf(
			`SELECT MEDIAN(value) FROM cloudData WHERE tagName='%v' AND time >= now() - %v AND time <= now() - %v + %v GROUP BY time(%v)`,
			sensorTagName,
			selectTime,
			selectTime,
			diffTime,
			groupTime,
		)
	}
	fmt.Println(query)

	res, err := influx.Query(query)
	if err != nil {
		return nil, err
	}

	columns := make([]string, 0, 10)
	xColumn := make([]int64, 0, 10)
	yColumn := make([]float64, 0, 10)

	for _, result := range res.Results {
		for _, series := range result.Series {
			for index, col := range series.Columns {
				if index != 0 {
					columns = append(columns, col)
				}

				for _, val := range series.Values {
					value := val[index]

					if index == 0 {
						parsedTime, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", value))
						if err != nil {
							continue
						}

						xColumn = append(xColumn, parsedTime.Unix())
					} else {
						resValue := 0.0
						if value != nil {
							f, err := getFloat(value)
							if err == nil {
								resValue = f
							}
						}

						yColumn = append(yColumn, resValue)
					}
				}
			}
		}
	}

	outputResult := &singleGraphData{
		XColumn: xColumn,
		YColumn: yColumn,
	}

	return outputResult, nil
}

func (influx *Influx) GetMultipleTagsData(tagNames []string, selectTime string, diffTime string, groupTime string) ResultGraphData {
	xColumn := make([]interface{}, 0, 10)

	columns := make([][]interface{}, 0, 10)
	for _, tagName := range tagNames {
		res, err := influx.GetSensorMedianData(tagName, selectTime, diffTime, groupTime)
		if err != nil {
			continue
		}

		if len(xColumn) == 0 && len(res.XColumn) > 0 {
			xColumn = append(xColumn, "x")
			for _, xCol := range res.XColumn {
				xColumn = append(xColumn, xCol)
			}
		}

		yColumn := make([]interface{}, 0, 10)
		yColumn = append(yColumn, tagName)
		for _, yCol := range res.YColumn {
			yColumn = append(yColumn, yCol)
		}
		columns = append(columns, yColumn)
	}

	resultColumns := make([][]interface{}, 0, 10)
	resultColumns = append(resultColumns, xColumn)
	for _, column := range columns {
		resultColumns = append(resultColumns, column)
	}

	return ResultGraphData{
		Columns: resultColumns,
	}
}

func (influx *Influx) GetLatestSensorValue(sensorId string) (int64, float64) {
	res, err := influx.Query(
		fmt.Sprintf(
			`SELECT last(value) FROM cloudData WHERE "tagName"='%s'`,
			sensorId,
		),
	)

	value := 0.0
	date := int64(0)
	if err == nil {
		for _, result := range res.Results {
			for _, series := range result.Series {
				if len(series.Values) > 0 && len(series.Values[0]) > 1 {
					formattedValue, err := getFloat(series.Values[0][1])
					if err == nil {
						value = formattedValue
					}

					layout := "2006-01-02T15:04:05Z"
					t, err := time.Parse(layout, fmt.Sprintf("%v", series.Values[0][0]))
					if err == nil {
						date = t.Unix()
					}
				}
			}
		}
	}

	return date, value
}

var floatType = reflect.TypeOf(float64(0))
var stringType = reflect.TypeOf("")

func getFloat(unk interface{}) (float64, error) {
	switch i := unk.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		return strconv.ParseFloat(i, 64)
	default:
		v := reflect.ValueOf(unk)
		v = reflect.Indirect(v)
		if v.Type().ConvertibleTo(floatType) {
			fv := v.Convert(floatType)
			return fv.Float(), nil
		} else if v.Type().ConvertibleTo(stringType) {
			sv := v.Convert(stringType)
			s := sv.String()
			return strconv.ParseFloat(s, 64)
		} else {
			return math.NaN(), fmt.Errorf("Can't convert %v to float64", v.Type())
		}
	}
}

func getInt(unk interface{}) (int64, error) {
	switch i := unk.(type) {
	case float64:
		return int64(i), nil
	case float32:
		return int64(i), nil
	case int64:
		return i, nil
	case int32:
		return int64(i), nil
	case int:
		return int64(i), nil
	case uint64:
		return int64(i), nil
	case uint32:
		return int64(i), nil
	case uint:
		return int64(i), nil
	case string:
		return strconv.ParseInt(i, 0, 64)
	default:
		v := reflect.ValueOf(unk)
		v = reflect.Indirect(v)
		if v.Type().ConvertibleTo(floatType) {
			fv := v.Convert(floatType)
			return fv.Int(), nil
		} else if v.Type().ConvertibleTo(stringType) {
			sv := v.Convert(stringType)
			s := sv.String()
			return strconv.ParseInt(s, 0, 64)
		} else {
			return 0, fmt.Errorf("Can't convert %v to float64", v.Type())
		}
	}
}

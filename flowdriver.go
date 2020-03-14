package flowdriver

import (
	"FlowDriver/flowerror"
	"errors"
	log "github.com/Delisa-sama/logger"
	"net/http"
	"reflect"
	"strconv"
)

type Invoker interface {
	Invoke() (status int, err flowerror.FlowError)
}

const (
	INPUT  = "In"     // имя поля входных значений обработчика
	OUTPUT = "Out"    // имя поля выходных значений обработчика
	INVOKE = "Invoke" // имя метода, исполняющего логику обработчика
)

const (
	STATUS = iota // индекс статуса в возвращаемых значениях Invoke
	ERROR         // индекс ошибки
)

type IntField = int64
type FloatField = float64
type StringField = string
type UintField = uint64
type BoolField = bool

// Провалидировать поля структуры на корректность типов полей
func validateInStruct(in reflect.Value) error {
	if in.Kind() != reflect.Struct {
		return errors.New("input value must be struct")
	}

	for i := 0; i < in.NumField(); i++ {
		// пропускаем приватные поля структуры
		if !in.Field(i).CanSet() {
			continue
		}

		kind := in.Field(i).Kind()
		switch kind {
		case reflect.Int64, reflect.Uint64, reflect.Float64, reflect.Bool, reflect.String:
			continue
		default:
			return errors.New("unsupported field type. " + in.Type().Field(i).Name + " => " + kind.String())
		}
	}

	return nil
}

// FlowDriver - обертка над http.HandlerFunc, предоставляющая валидацию входных и выходных значений обработчика.
// Принимает обработчик - структура, имплементирующая интерфейс Invoker.
// Обработчик должен иметь поля In и Out - которые в свою очередь являются структурами (не указателями на структуру)
// Поля структур In и Out должны быть публичными и описываться типами пакета flowdriver, например, flowdriver.IntField
// При несоответствии типов, вызывает panic
func FlowDriver(in Invoker) http.HandlerFunc {
	logger := log.GetLogger()

	original := reflect.ValueOf(in)
	handlerName := reflect.TypeOf(in).String()
	if original.Kind() != reflect.Struct {
		panic("[" + handlerName + "] Invalid FlowDriver input. Got: " + original.Kind().String() + ". Expected: " + reflect.Struct.String())
	}

	originalIn := original.FieldByName(INPUT)
	if originalIn.Kind() != reflect.Struct {
		panic("[" + handlerName + "] Invalid Invoker struct. Field " + INPUT + ". Got: " + originalIn.Kind().String() + ". Expected: " + reflect.Struct.String())
	}
	// проверяем что типы полей входной структуры корректны
	if err := validateInStruct(originalIn); err != nil {
		panic("[" + handlerName + "] Invalid Invoker struct. " + err.Error())
	}

	// пока эксперемент, но по идее будет возвращаться http.HandlerFunc
	return func(w http.ResponseWriter, r *http.Request) {
		copied := reflect.New(original.Type()).Elem() // копия структуры, в которой будут хранится значения и у которого будем вызывать Invoke
		copiedIn := copied.FieldByName(INPUT)

		out := copied.FieldByName(OUTPUT)
		// проверяем что типы полей входной структуры корректны
		if err := validateInStruct(out); err != nil {
			panic("[" + handlerName + "] Invalid Invoker struct. " + err.Error())
		}

		for i := 0; i < originalIn.NumField(); i++ {
			originalInField := originalIn.Field(i)
			copyInField := copiedIn.Field(i)
			// пропускаем приватные поля структуры
			if !copyInField.CanSet() {
				continue
			}

			fieldName := originalIn.Type().Field(i).Name
			value := r.FormValue(fieldName) // TODO: возможность задать имя параметра через тэг структуры
			if len(value) == 0 {
				logger.Errorf("[%s] Input field is empty => %s", handlerName, fieldName)
				_ = WriteJSONError(w, flowerror.New("EMPTY_INPUT", "Missing input field: "+fieldName), http.StatusBadRequest)
				return
			}

			var conv interface{}
			var err interface{}
			switch originalInField.Kind() {
			case reflect.String:
				conv = value
			case reflect.Bool:
				conv, err = strconv.ParseBool(value)
				conv = conv.(bool)
			case reflect.Int64:
				conv, err = strconv.ParseInt(value, 10, 64)
				if err == nil && copyInField.OverflowInt(conv.(int64)) {
					err = errors.New("integer overflow detected")
				}
				conv = conv.(int64)
			case reflect.Float64:
				conv, err = strconv.ParseFloat(value, 64)
				if err == nil && copyInField.OverflowFloat(conv.(float64)) {
					err = errors.New("float overflow detected")
				}
				conv = conv.(float64)
			case reflect.Uint64:
				conv, err = strconv.ParseUint(value, 10, 64)
				if err == nil && copyInField.OverflowUint(conv.(uint64)) {
					err = errors.New("uint overflow detected")
				}
				conv = conv.(uint64)
			default:
				err = errors.New("unsupported field type")
			}

			if err != nil {
				logger.Errorf("[%s] Failed to parse input field %s = %s => %s\n Error: %s", handlerName, fieldName, value, originalInField.Kind(), err)
				_ = WriteJSONError(w, flowerror.New("INVALID_FIELD_TYPE", "Invalid input"), http.StatusBadRequest)
				return
			}

			copyInField.Set(reflect.ValueOf(conv))
		}

		result := copied.MethodByName(INVOKE).Call([]reflect.Value{})

		statusInterface := result[STATUS].Interface()
		if statusInterface == nil {
			logger.Errorf("[%s] Invoked statusInterface cannot be nil", handlerName)
			_ = WriteJSONError(w, flowerror.New("BAD_STATUS", "Internal server error. Bad status."), http.StatusBadGateway)
			return
		}
		var status = statusInterface.(int)

		errInterface := result[ERROR].Interface()
		if errInterface != nil {
			var err = errInterface.(flowerror.FlowError)
			logger.Errorf("[%s] %s", handlerName, err)
			_ = WriteJSONError(w, err, status)
			return
		}

		outInterface := out.Interface()
		if outInterface == nil {
			logger.Errorf("[%s] Out struct cannot be nil", handlerName)
			_ = WriteJSONError(w, flowerror.New("BAD_OUTPUT", "Internal server error. Bad output."), http.StatusBadGateway)
			return
		}

		_ = WriteJSONResponse(w, outInterface, status)
	}
}

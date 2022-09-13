package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jaegertracing/jaeger/model"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"
)

type JaegerUIModel struct {
	Data []JaegerUITrace `json:"data"`
}

type JaegerUITrace struct {
	TraceId   string                     `json:"traceID"`
	Spans     []JaegerUISpan             `json:"spans"`
	Processes map[string]JaegerUIProcess `json:"processes"`
}

type JaegerUISpan struct {
	Duration      int64               `json:"duration"`
	OperationName string              `json:"operationName"`
	References    []JaegerUIReference `json:"references"`
	StartTime     int64               `json:"startTime"`
	SpanId        string              `json:"spanID"`
	Tags          []JaegerUITag       `json:"tags"`
	TraceId       string              `json:"traceID"`
	Process       string              `json:"processID"`
}

type JaegerUIReference struct {
	RefType string `json:"refType"`
	TraceId string `json:"traceID"`
	SpanId  string `json:"spanId"`
}

type JaegerUITag struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type JaegerUIProcess struct {
	ServiceName string        `json:"serviceName"`
	Tags        []JaegerUITag `json:"tags"`
}

type CHSpanModel struct {
	TraceId       string        `json:"trace_id"`
	SpanId        string        `json:"span_id"`
	OperationName string        `json:"operation_name"`
	References    []CHReference `json:"references"`
	StartTime     string        `json:"start_time"`
	Duration      int64         `json:"duration"`
	Tags          []CHTag       `json:"tags"`
	Logs          []string      `json:"logs"`
	Process       CHProcess     `json:"process"`
}

type CHReference struct {
	TraceId string `json:"trace_id"`
	SpanId  string `json:"span_id"`
}

type CHTag struct {
	Key   string `json:"key"`
	Value string `json:"v_str"`
}

type CHProcess struct {
	ServiceName string  `json:"service_name"`
	Tags        []CHTag `json:"tags"`
}

type CHIndexModel struct {
	Timestamp  string
	TraceId    string
	Service    string
	Operation  string
	DurationUs int64
	//Tags []string (this is an array transformed as string)
	Tags string
}

func main() {

	service := ""
	file := ""

	// Parse command line flags
	flag.StringVar(&service, "service", "", "the service name to import as")
	flag.StringVar(&file, "file", "", "the file you exported for the service")
	flag.Parse()

	if service == "" || file == "" {
		log.Fatal("Please provide a value for -service and -file (these will be imported)")
	}

	//jaegerUiTracesFile := "./traces-console-ui.json"
	//jaegerUiTracesFile := "./single-trace-console-ui-test.json"

	log.Println("Generating ClickHouse INSERT statements for service " + service + " via file " + file + "...")

	// Let's first read the traces file
	//content, err := ioutil.ReadFile(jaegerUiTracesFile)
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	// Now let's unmarshall the data into `payload`
	var payload JaegerUIModel
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// Iterate thru the content
	for _, trace := range payload.Data {
		//fmt.Printf("%v\n", trace)

		for _, span := range trace.Spans {
			var serialized []byte
			//serialized, err = json.Marshal(span)
			spanModel := convertUISpanToCHSpanModel(span, trace.Processes)
			serialized, err = json.Marshal(spanModel)
			if err != nil {
				log.Fatal("Error during Marshalling of span: ", err)
			}

			fmt.Printf("INSERT INTO jaeger_spans_local (timestamp, traceID, model) VALUES ('%s', '%s', '%s');\n", strconv.Itoa(int(span.StartTime/int64(time.Millisecond))), span.TraceId, serialized)
			//fmt.Printf("What we'll insert: %s, %s, %s", strconv.Itoa(time.Unix(span.StartTime/int64(time.Millisecond), 0)), span.TraceId, serialized)

			// Let's also insert into index
			m := convertCHSpanModelToIndexModel(spanModel, service)
			fmt.Printf("INSERT INTO jaeger_index_local (timestamp, traceID, service, operation, durationUs, tags) VALUES ('%s', '%s', '%s', '%s', '%s', %s);\n", strconv.Itoa(int(span.StartTime/int64(time.Millisecond))), span.TraceId, m.Service, m.Operation, strconv.Itoa(int(m.DurationUs)), m.Tags)

		}
		break
	}

	log.Println("Done!")
}

// Use jaeger lib at https://github.com/jaegertracing/jaeger/blob/c1bb2946e670129314bf88d9730d8ed7566766d4/model/ids.go
func convertToTraceIdModel(sTrace string) string {
	t, _ := model.TraceIDFromString(sTrace)

	// MarshalJSON adds quotes to it string
	// so we have to make sure to trim those
	b64t, _ := t.MarshalJSON()
	s := string(b64t[1 : len(b64t)-1])

	return s
}

// Use jaeger lib at https://github.com/jaegertracing/jaeger/blob/c1bb2946e670129314bf88d9730d8ed7566766d4/model/ids.go
func convertToSpanIdModel(sSpan string) string {
	span, _ := model.SpanIDFromString(sSpan)

	// MarshalJSON adds quotes to it string
	// so we have to make sure to trim those
	b64s, _ := span.MarshalJSON()
	s := string(b64s[1 : len(b64s)-1])

	return s
}

func convertCHSpanModelToIndexModel(sm CHSpanModel, svc string) CHIndexModel {
	var im CHIndexModel

	im.Timestamp = sm.StartTime
	im.TraceId = sm.TraceId
	im.Service = svc
	im.Operation = sm.OperationName
	im.DurationUs = sm.Duration

	var tags []string
	for _, tag := range sm.Tags {
		rawkv := "'" + tag.Key + "=" + sanitizeTagValue(tag.Value) + "'"
		//rawkv := tag.Key+"="+tag.Value
		tags = append(tags, rawkv)
	}
	//im.Tags = tags
	im.Tags = "[" + strings.Join(tags, ",") + "]"

	return im
}

func sanitizeTagValue(v string) string {
	s := v

	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	return s
}

func convertUISpanToCHSpanModel(s JaegerUISpan, procMap map[string]JaegerUIProcess) CHSpanModel {
	var c CHSpanModel

	c.TraceId = convertToTraceIdModel(s.TraceId)
	c.SpanId = convertToSpanIdModel(s.SpanId)
	c.OperationName = s.OperationName
	//c.StartTime = time.Unix(s.StartTime/int64(time.Millisecond), 0).UTC().Format(time.RFC3339)
	//c.StartTime = time.Unix(s.StartTime/int64(time.Millisecond), 6).UTC().Format(time.RFC3339Nano)
	c.StartTime = time.Unix(s.StartTime/int64(time.Millisecond), 1).UTC().Format(time.RFC3339Nano)
	c.Duration = s.Duration
	//c.Logs = ""
	c.Process.ServiceName = procMap[s.Process].ServiceName
	c.Process.Tags = convertUITagToModelTag(procMap[s.Process].Tags)

	var refs []CHReference
	for _, sref := range s.References {
		r := CHReference{
			TraceId: convertToTraceIdModel(sref.TraceId),
			SpanId:  convertToSpanIdModel(sref.SpanId),
		}
		refs = append(refs, r)
	}
	c.References = refs

	c.Tags = convertUITagToModelTag(s.Tags)

	return c
}

func convertUITagToModelTag(uTags []JaegerUITag) []CHTag {
	var tags []CHTag
	for _, utag := range uTags {
		valAsString := sanitizeTagValue(fmt.Sprintf("%v", utag.Value))

		t := CHTag{
			Key:   utag.Key,
			Value: valAsString,
		}
		tags = append(tags, t)
	}

	return tags
}

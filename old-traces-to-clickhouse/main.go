package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	TraceId string         `json:"traceID"`
	Spans   []JaegerUISpan `json:"spans"`
}

type JaegerUISpan struct {
	Duration      int64               `json:"duration"`
	OperationName string              `json:"operationName"`
	References    []JaegerUIReference `json:"references"`
	StartTime     int64               `json:"startTime"`
	SpanId        string              `json:"spanID"`
	Tags          []JaegerUITag       `json:"tags"`
	TraceId       string              `json:"traceID"`
}

type JaegerUIReference struct {
	RefType string `json:"refType"`
	TraceId string `json:"traceID"`
	SpanId  string `json:"spanId"`
}

type JaegerUITag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type CHSpanModel struct {
	TraceId       string        `json:"trace_id"`
	SpanId        string        `json:"span_id"`
	OperationName string        `json:"operation_name"`
	References    []CHReference `json:"references"`
	StartTime     string        `json:"start_time"`
	Duration      int64         `json:"duration"`
	Tags          []CHTag       `json:"tags"`
}

type CHReference struct {
	TraceId string `json:"trace_id"`
	SpanId  string `json:"span_id"`
}

type CHTag struct {
	Key   string `json:"key"`
	Value string `json:"v_str"`
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
			spanModel := convertUISpanToCHSpanModel(span)
			serialized, err = json.Marshal(spanModel)
			if err != nil {
				log.Fatal("Error during Marshalling of span: ", err)
			}

			fmt.Printf("INSERT INTO jaeger_spans_local (timestamp, traceID, model) VALUES ('%s', '%s', '%s');\n", strconv.Itoa(int(span.StartTime/int64(time.Millisecond))), span.TraceId, serialized)
			//fmt.Printf("What we'll insert: %s, %s, %s", strconv.Itoa(time.Unix(span.StartTime/int64(time.Millisecond), 0)), span.TraceId, serialized)

			// Let's also insert into index
			m := convertCHSpanModelToIndexModel(spanModel, service)
			fmt.Printf("INSERT INTO jaeger_index_local (timestamp, traceID, service, operation, durationUs, tags) VALUES ('%s', '%s', '%s', '%s', '%s', %s);\n", strconv.Itoa(int(span.StartTime/int64(time.Millisecond))), m.TraceId, m.Service, m.Operation, strconv.Itoa(int(m.DurationUs)), m.Tags)
		}
		break
	}

	log.Println("Done!")
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
		rawkv := "'" + tag.Key + "=" + tag.Value + "'"
		//rawkv := tag.Key+"="+tag.Value
		tags = append(tags, rawkv)
	}
	//im.Tags = tags
	im.Tags = "[" + strings.Join(tags, ",") + "]"

	return im
}

func convertUISpanToCHSpanModel(s JaegerUISpan) CHSpanModel {
	var c CHSpanModel

	c.TraceId = s.TraceId
	c.SpanId = s.SpanId
	c.OperationName = s.OperationName
	c.StartTime = time.Unix(s.StartTime/int64(time.Millisecond), 0).UTC().Format(time.RFC3339)
	c.Duration = s.Duration

	var refs []CHReference
	for _, sref := range s.References {
		r := CHReference{
			TraceId: sref.TraceId,
			SpanId:  sref.SpanId,
		}
		refs = append(refs, r)
	}
	c.References = refs

	var tags []CHTag
	for _, stag := range s.Tags {
		t := CHTag{
			Key:   stag.Key,
			Value: stag.Value,
		}
		tags = append(tags, t)
	}
	c.Tags = tags

	return c
}

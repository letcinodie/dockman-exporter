package main

import (
	"strings"
	"encoding/json"
	"io"
	"net"
    "net/http"
	"os"
	"log"
	"context"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

//Variables to store Docker/Podman API json
type Json map[string]interface{}
type DockerImages []Json

//Prometheus specific struct
type dockerImageCollector struct {
	dockerImageMetric *prometheus.Desc
}

//Find if docker or podman is used
func findSocket() string {
	var socket string
	sockets := [3]string{"/var/run/podman.sock", "/var/run/docker.sock", "/var/run/user/1000/podman/podman.sock"}
	
	for _, socket = range sockets {
		log.Println(socket)
		if _, err := os.Stat(socket); err == nil {
			return socket
		} 
	}
	log.Fatal("No Podman or Docker socket found!")
	return socket
}

//Connect to socket and trigger.
//At the moment we only request /images/json
//To be expanded
func getImageList(socket string) DockerImages {
	uri := "/images/json"

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}

	var response *http.Response
	var err error
	//List images
	response, err = httpc.Get("http://unix" + uri)

	if err != nil {
		log.Fatal(err)
	}
	var imagesJson DockerImages
	if response.StatusCode == 200 {
		

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		
		if err := json.Unmarshal([]byte(body), &imagesJson); err != nil {
        	log.Fatal(err)
    	}
		
	}

	return imagesJson
}

func getHostname(socket string) string {
	uri := "/info"

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}

	var response *http.Response
	var err error
	var hostname string
	//List images
	response, err = httpc.Get("http://unix" + uri)

	if err != nil {
		log.Fatal(err)
	}
	var infoJson Json
	if response.StatusCode == 200 {
		

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		
		if err := json.Unmarshal([]byte(body), &infoJson); err != nil {
        	log.Fatal(err)
    	}
		hostname = infoJson["Name"].(string)
		
	}

	return hostname
}


//I barely understand how this works
func newDockerImageCollector() *dockerImageCollector {
	return &dockerImageCollector{
		dockerImageMetric: prometheus.NewDesc("docker_image_size", "Docker Image Size in bytes.",
			[]string{"hostname","docker_image_name", "docker_image_tag","docker_image_id"}, nil,
		),
	}
}

func (collector *dockerImageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.dockerImageMetric
}

func (collector *dockerImageCollector) Collect(ch chan<- prometheus.Metric) {
	var imageName string
	var imageTag string
	var hostname string
	socket := findSocket()
	json := getImageList(socket)
	hostname = getHostname(socket)
	
	for _, image := range json {
		if len(image["RepoTags"].([]interface{})) != 0 {
			img := image["RepoTags"].([]interface{})[0].(string)
			imageName = strings.Split(img, ":")[0]
			imageTag = strings.Split(img, ":")[1]

		} else if len(image["RepoDigests"].([]interface{})) != 0 {
			img := image["RepoDigests"].([]interface{})[0].(string)
			imageName = strings.Split(img, "@")[0]
			imageTag = "none"

		}
		imageId := string(image["Id"].(string))
		imageSize := float64(image["Size"].(float64))
		m1 := prometheus.MustNewConstMetric(collector.dockerImageMetric, prometheus.GaugeValue, imageSize, hostname, imageName, imageTag, imageId )
		ch <- m1
	}
}

func main() {
	dockerImageSize := newDockerImageCollector()
	prometheus.MustRegister(dockerImageSize)

    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9910", nil)
	log.Fatal(http.ListenAndServe(":9910", nil))
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func (s *Server) HandleHeathCheck(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHeathCheck")
}

func (s *Server) HandleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHelloWorld")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello Dapr K8s!"))
}

func (s *Server) HandleCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleCreate")

	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	log.Println(req)
	result, err := s.kubeClient.CreateAppDeploy(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

type DeleteRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (s *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDelete")
	var req DeleteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleError(w, err)
	}
	log.Println(req)

	result, err := s.kubeClient.DeleteAppDeploy(req.Namespace, req.Name)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDisconnect")
	var req DeleteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleError(w, err)

	}
	log.Println(req)
	result, err := s.kubeClient.DisconnectRedis(req.Namespace, req.Name)
	if err != nil {
		HandleError(w, err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func HandleError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// mutating incoming pods
func handleMutation(w http.ResponseWriter, r *http.Request) {
	var admissionReviewReq admissionv1.AdmissionReview
	var admissionReviewResp admissionv1.AdmissionReview

	// request recorded 
	if err := json.NewDecoder(r.Body).Decode(&admissionReviewReq); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode request: %v", err), http.StatusBadRequest)
		return
	}

	admissionReviewResp.TypeMeta = admissionv1.SchemeGroupVersion.WithKind("AdmissionReview")
	admissionReviewResp.Response = &admissionv1.AdmissionResponse{
		UID:     admissionReviewReq.Request.UID,
		Allowed: true,
	}

	// fetch the pods fromt the request
	raw := admissionReviewReq.Request.Object.Raw
	pod := corev1.Pod{}
	if err := json.Unmarshal(raw, &pod); err != nil {
		http.Error(w, fmt.Sprintf("Could not unmarshal pod: %v", err), http.StatusInternalServerError)
		return
	}

	// add a label apply mutation
	patch := []jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/metadata/labels/mutated",
			Value:     "true",
		},
	}

	patchBytes, _ := json.Marshal(patch)
	admissionReviewResp.Response.Patch = patchBytes
	patchType := admissionv1.PatchTypeJSONPatch
	admissionReviewResp.Response.PatchType = &patchType

	// response sent
	respBytes, _ := json.Marshal(admissionReviewResp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// validates incoming podss
func handleValidation(w http.ResponseWriter, r *http.Request) {
	var admissionReviewReq admissionv1.AdmissionReview
	var admissionReviewResp admissionv1.AdmissionReview

	
	if err := json.NewDecoder(r.Body).Decode(&admissionReviewReq); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode request: %v", err), http.StatusBadRequest)
		return
	}

	admissionReviewResp.TypeMeta = admissionv1.SchemeGroupVersion.WithKind("AdmissionReview")
	admissionReviewResp.Response = &admissionv1.AdmissionResponse{
		UID: admissionReviewReq.Request.UID,
	}

	// fetch pod from  request
	raw := admissionReviewReq.Request.Object.Raw
	pod := corev1.Pod{}
	if err := json.Unmarshal(raw, &pod); err != nil {
		http.Error(w, fmt.Sprintf("Could not unmarshal pod: %v", err), http.StatusInternalServerError)
		return
	}

	// here pods running as root aare blocked
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
			admissionReviewResp.Response.Allowed = false
			admissionReviewResp.Response.Result = &metav1.Status{
				Message: "Running as root is not allowed!",
			}
			break
		} else {
			admissionReviewResp.Response.Allowed = true
		}
	}

	
	respBytes, _ := json.Marshal(admissionReviewResp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func main() {
	http.HandleFunc("/mutate", handleMutation)
	http.HandleFunc("/validate", handleValidation)

	port := "8443"
	fmt.Println("Starting webhook server on port", port)

	if err := http.ListenAndServeTLS(":"+port, "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

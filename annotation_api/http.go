package annotation_api

import (
	"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/wndhydrnt/proxym/log"
	"io/ioutil"
	"net/http"
)

type AnnotationListItem struct {
	Annotation *Annotation `json:"annotation"`
	Link       string      `json:"link"`
}

type Http struct {
	zkCon *zk.Conn
}

func (h *Http) createAnnotation(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	serviceId := params.Get(":serviceId")
	serviceZkPath := zookeeperPath + "/" + serviceId

	w.Header().Set("Access-Control-Allow-Origin", "*")

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading request body: '%s'", err), http.StatusBadRequest)
		return
	}

	annotation := &Annotation{}
	err = json.Unmarshal(data, annotation)
	if err != nil {
		log.ErrorLog.Error("Error reading from Zookeeper: '%s'", err)
		http.Error(w, fmt.Sprintf("Error parsing JSON: '%s'", err), http.StatusBadRequest)
		return
	}

	present, _, err := h.zkCon.Exists(serviceZkPath)
	if err != nil {
		log.ErrorLog.Error("Error reading from Zookeeper: '%s'", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if present {
		zkData, _, err := h.zkCon.Get(serviceZkPath)
		if err != nil {
			log.ErrorLog.Error("Error reading from Zookeeper: '%s'", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		oldA := &Annotation{}
		json.Unmarshal(zkData, oldA)
		if oldA.Config != annotation.Config || oldA.ApplicationProtocol != annotation.ApplicationProtocol || oldA.ProxyPath != annotation.ProxyPath || !compareDomains(oldA.Domains, annotation.Domains) {
			newData, err := json.Marshal(annotation)
			if err != nil {
				log.ErrorLog.Error("Error marshalling Annotation: '%s'", err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			_, err = h.zkCon.Set(serviceZkPath, newData, int32(-1))
			if err != nil {
				log.ErrorLog.Error("Error updating zNode '%s': '%s'", serviceZkPath, err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
	} else {
		newData, err := json.Marshal(annotation)
		if err != nil {
			log.ErrorLog.Error("Error marshalling Annotation: '%s'", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		_, err = h.zkCon.Create(serviceZkPath, newData, int32(0), zk.WorldACL(zk.PermAll))
		if err != nil {
			log.ErrorLog.Error("Error creating zNode '%s': '%s'", serviceZkPath, err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Http) deleteAnnotation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	params := r.URL.Query()
	aId := params.Get(":serviceId")

	err := h.zkCon.Delete(zookeeperPath+"/"+aId, int32(-1))
	if err != nil {
		log.ErrorLog.Error("Error deleting annotation ID '%s': '%s'", aId, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Http) listAnnotations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	annotationList := make([]*AnnotationListItem, 0)

	annotationIds, _, err := h.zkCon.Children(zookeeperPath)
	if err != nil {
		log.ErrorLog.Error("Error reading annotation IDs: '%s'", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	for _, aId := range annotationIds {
		path := zookeeperPath + "/" + aId
		annotationData, _, err := h.zkCon.Get(path)
		if err != nil {
			log.ErrorLog.Error("Error reading data of annotation ID '%s': '%s'", aId, err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		annotation := &Annotation{}
		err = json.Unmarshal(annotationData, annotation)
		if err != nil {
			log.ErrorLog.Error("Error unmarshalling data of annotation ID '%s': '%s'", aId, err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		listItem := &AnnotationListItem{
			Annotation: annotation,
			Link:       httpPrefix + "/" + aId,
		}

		annotationList = append(annotationList, listItem)
	}

	anntationListData, err := json.Marshal(annotationList)
	if err != nil {
		log.ErrorLog.Error("Error marshalling list of annotations: '%s'", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Write(anntationListData)
}

func (h *Http) optionsAnnotation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE,POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func NewHttp(zkCon *zk.Conn) *Http {
	return &Http{zkCon: zkCon}
}

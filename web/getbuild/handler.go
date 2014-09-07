package getbuild

import (
	"html/template"
	"net/http"

	"github.com/concourse/atc/builds"
	"github.com/concourse/atc/config"
	"github.com/concourse/atc/db"
	"github.com/pivotal-golang/lager"
)

type handler struct {
	logger lager.Logger

	jobs     config.Jobs
	db       db.DB
	template *template.Template
}

func NewHandler(logger lager.Logger, jobs config.Jobs, db db.DB, template *template.Template) http.Handler {
	return &handler{
		logger: logger,

		jobs:     jobs,
		db:       db,
		template: template,
	}
}

type TemplateData struct {
	Job    config.Job
	Builds []builds.Build

	Build   builds.Build
	Inputs  []db.BuildInput
	Outputs []db.BuildOutput

	Abortable bool
}

func (handler *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue(":job")
	if len(jobName) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	buildName := r.FormValue(":build")
	if len(buildName) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	job, found := handler.jobs.Lookup(jobName)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log := handler.logger.Session("get-build", lager.Data{
		"job":   job.Name,
		"build": buildName,
	})

	build, err := handler.db.GetJobBuild(jobName, buildName)
	if err != nil {
		log.Error("get-build-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	inputs, outputs, err := handler.db.GetBuildResources(build.ID)
	if err != nil {
		log.Error("failed-to-get-build-resources", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bs, err := handler.db.GetAllJobBuilds(jobName)
	if err != nil {
		log.Error("get-all-builds-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var abortable bool
	switch build.Status {
	case builds.StatusPending, builds.StatusStarted:
		abortable = true
	default:
		abortable = false
	}

	templateData := TemplateData{
		Job:    job,
		Builds: bs,

		Build:     build,
		Inputs:    inputs,
		Outputs:   outputs,
		Abortable: abortable,
	}

	err = handler.template.Execute(w, templateData)
	if err != nil {
		log.Fatal("failed-to-execute-template", err, lager.Data{
			"template-data": templateData,
		})
	}
}

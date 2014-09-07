package db_test

import (
	"fmt"

	Builds "github.com/concourse/atc/builds"
	"github.com/concourse/atc/config"
	. "github.com/concourse/atc/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func itIsADB() {
	BeforeEach(func() {
		err := db.RegisterJob("some-job")
		Ω(err).ShouldNot(HaveOccurred())

		err = db.RegisterJob("some-job")
		Ω(err).ShouldNot(HaveOccurred())

		err = db.RegisterResource("some-resource")
		Ω(err).ShouldNot(HaveOccurred())

		err = db.RegisterResource("some-resource")
		Ω(err).ShouldNot(HaveOccurred())

		err = db.RegisterJob("some-other-job")
		Ω(err).ShouldNot(HaveOccurred())

		err = db.RegisterResource("some-other-resource")
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("works", func() {
		builds, err := db.GetAllJobBuilds("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(builds).Should(BeEmpty())

		_, err = db.GetCurrentBuild("some-job")
		Ω(err).Should(HaveOccurred())

		build, err := db.CreateBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(build.ID).ShouldNot(BeZero())
		Ω(build.JobName).Should(Equal("some-job"))
		Ω(build.Name).Should(Equal("1"))
		Ω(build.Status).Should(Equal(Builds.StatusPending))

		gotBuild, err := db.GetBuild(build.ID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(gotBuild).Should(Equal(build))

		pending, err := db.CreateBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(pending.ID).ShouldNot(BeZero())
		Ω(pending.ID).ShouldNot(Equal(build.ID))
		Ω(pending.Name).Should(Equal("2"))
		Ω(pending.Status).Should(Equal(Builds.StatusPending))

		nextPending, _, err := db.GetNextPendingBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(nextPending).Should(Equal(build))

		currentBuild, err := db.GetCurrentBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(currentBuild).Should(Equal(build))

		scheduled, err := db.ScheduleBuild("some-job", build.Name, false)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(scheduled).Should(BeTrue())

		nextPending, _, err = db.GetNextPendingBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(nextPending).Should(Equal(pending))

		build, err = db.GetCurrentBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(build.Name).Should(Equal("1"))
		Ω(build.Status).Should(Equal(Builds.StatusPending))

		started, err := db.StartBuild("some-job", build.Name, "some-abort-url")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(started).Should(BeTrue())

		build, err = db.GetCurrentBuild("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(build.Name).Should(Equal("1"))
		Ω(build.Status).Should(Equal(Builds.StatusStarted))
		Ω(build.AbortURL).Should(Equal("some-abort-url"))

		builds, err = db.GetAllJobBuilds("some-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(builds).Should(HaveLen(2))
		Ω(builds[0].Name).Should(Equal(pending.Name))
		Ω(builds[0].JobName).Should(Equal("some-job"))
		Ω(builds[0].Status).Should(Equal(Builds.StatusPending))
		Ω(builds[1].Name).Should(Equal(build.Name))
		Ω(builds[1].JobName).Should(Equal("some-job"))
		Ω(builds[1].Status).Should(Equal(Builds.StatusStarted))

		err = db.SaveBuildStatus(build.ID, Builds.StatusSucceeded)
		Ω(err).ShouldNot(HaveOccurred())

		build, err = db.GetJobBuild("some-job", build.Name)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(build.ID).ShouldNot(BeZero())
		Ω(build.Name).Should(Equal("1"))
		Ω(build.JobName).Should(Equal("some-job"))
		Ω(build.Status).Should(Equal(Builds.StatusSucceeded))

		otherBuild, err := db.CreateBuild("some-other-job")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(otherBuild.ID).ShouldNot(BeZero())
		Ω(otherBuild.Name).Should(Equal("1"))
		Ω(otherBuild.Status).Should(Equal(Builds.StatusPending))

		build, err = db.GetJobBuild("some-other-job", otherBuild.Name)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(build.Name).Should(Equal("1"))
		Ω(build.Status).Should(Equal(Builds.StatusPending))

		started, err = db.StartBuild("some-other-job", build.Name, "some-other-abort-url")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(started).Should(BeTrue())

		abortURL, err := db.AbortBuild(build.ID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(abortURL).Should(Equal("some-other-abort-url"))

		log, err := db.BuildLog(build.ID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(log)).Should(Equal(""))

		err = db.AppendBuildLog(build.ID, []byte("some "))
		Ω(err).ShouldNot(HaveOccurred())

		err = db.AppendBuildLog(build.ID, []byte("log"))
		Ω(err).ShouldNot(HaveOccurred())

		log, err = db.BuildLog(build.ID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(log)).Should(Equal("some log"))
	})

	Describe("saving build inputs", func() {
		buildMetadata := []Builds.MetadataField{
			{
				Name:  "meta1",
				Value: "value1",
			},
			{
				Name:  "meta2",
				Value: "value2",
			},
		}

		vr1 := Builds.VersionedResource{
			Name:     "some-resource",
			Type:     "some-type",
			Source:   config.Source{"some": "source"},
			Version:  Builds.Version{"ver": "1"},
			Metadata: buildMetadata,
		}

		vr2 := Builds.VersionedResource{
			Name:    "some-other-resource",
			Type:    "some-type",
			Source:  config.Source{"some": "other-source"},
			Version: Builds.Version{"ver": "2"},
		}

		It("saves build's inputs and outputs as versioned resources", func() {
			build, err := db.CreateBuild("some-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(build.ID, vr1)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = db.GetJobBuildForInputs("some-job", Builds.VersionedResources{vr1, vr2})
			Ω(err).Should(HaveOccurred())

			err = db.SaveBuildInput(build.ID, vr2)
			Ω(err).ShouldNot(HaveOccurred())

			foundBuild, err := db.GetJobBuildForInputs("some-job", Builds.VersionedResources{vr1, vr2})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(foundBuild).Should(Equal(build))

			err = db.SaveBuildOutput(build.ID, vr1)
			Ω(err).ShouldNot(HaveOccurred())

			modifiedVR2 := vr2
			modifiedVR2.Version = Builds.Version{"ver": "3"}

			err = db.SaveBuildOutput(build.ID, modifiedVR2)
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(build.ID, vr2)
			Ω(err).ShouldNot(HaveOccurred())

			inputs, outputs, err := db.GetBuildResources(build.ID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(inputs).Should(ConsistOf([]BuildInput{
				{VersionedResource: vr1, FirstOccurrence: true},
				{VersionedResource: vr2, FirstOccurrence: true},
			}))
			Ω(outputs).Should(ConsistOf([]BuildOutput{
				{VersionedResource: modifiedVR2},
			}))

			duplicateBuild, err := db.CreateBuild("some-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(duplicateBuild.ID, vr1)
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(duplicateBuild.ID, vr2)
			Ω(err).ShouldNot(HaveOccurred())

			inputs, _, err = db.GetBuildResources(duplicateBuild.ID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(inputs).Should(ConsistOf([]BuildInput{
				{VersionedResource: vr1, FirstOccurrence: false},
				{VersionedResource: vr2, FirstOccurrence: false},
			}))

			newBuildInOtherJob, err := db.CreateBuild("some-other-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(newBuildInOtherJob.ID, vr1)
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(newBuildInOtherJob.ID, vr2)
			Ω(err).ShouldNot(HaveOccurred())

			inputs, _, err = db.GetBuildResources(newBuildInOtherJob.ID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(inputs).Should(ConsistOf([]BuildInput{
				{VersionedResource: vr1, FirstOccurrence: true},
				{VersionedResource: vr2, FirstOccurrence: true},
			}))
		})

		It("updates metadata of existing inputs resources", func() {
			build, err := db.CreateBuild("some-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildInput(build.ID, vr2)
			Ω(err).ShouldNot(HaveOccurred())

			inputs, _, err := db.GetBuildResources(build.ID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(inputs).Should(ConsistOf([]BuildInput{
				{VersionedResource: vr2, FirstOccurrence: true},
			}))

			withMetadata := vr2
			withMetadata.Metadata = buildMetadata

			err = db.SaveBuildInput(build.ID, withMetadata)
			Ω(err).ShouldNot(HaveOccurred())

			inputs, _, err = db.GetBuildResources(build.ID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(inputs).Should(ConsistOf([]BuildInput{
				{VersionedResource: withMetadata, FirstOccurrence: true},
			}))
		})

		It("can be done on build creation", func() {
			inputs := Builds.VersionedResources{vr1, vr2}

			pending, err := db.CreateBuildWithInputs("some-job", inputs)
			Ω(err).ShouldNot(HaveOccurred())

			foundBuild, err := db.GetJobBuildForInputs("some-job", inputs)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(foundBuild).Should(Equal(pending))

			nextPending, pendingInputs, err := db.GetNextPendingBuild("some-job")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(nextPending).Should(Equal(pending))
			Ω(pendingInputs).Should(Equal(inputs))
		})
	})

	Describe("saving versioned resources", func() {
		It("updates the latest versioned resource", func() {
			vr1 := Builds.VersionedResource{
				Name:    "some-resource",
				Type:    "some-type",
				Source:  config.Source{"some": "source"},
				Version: Builds.Version{"version": "1"},
				Metadata: []Builds.MetadataField{
					{Name: "meta1", Value: "value1"},
				},
			}

			vr2 := Builds.VersionedResource{
				Name:    "some-resource",
				Type:    "some-type",
				Source:  config.Source{"some": "source"},
				Version: Builds.Version{"version": "2"},
				Metadata: []Builds.MetadataField{
					{Name: "meta2", Value: "value2"},
				},
			}

			vr3 := Builds.VersionedResource{
				Name:    "some-resource",
				Type:    "some-type",
				Source:  config.Source{"some": "source"},
				Version: Builds.Version{"version": "3"},
				Metadata: []Builds.MetadataField{
					{Name: "meta3", Value: "value3"},
				},
			}

			err := db.SaveVersionedResource(vr1)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestVersionedResource("some-resource")).Should(Equal(vr1))

			err = db.SaveVersionedResource(vr2)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestVersionedResource("some-resource")).Should(Equal(vr2))

			err = db.SaveVersionedResource(vr3)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestVersionedResource("some-resource")).Should(Equal(vr3))
		})

		It("overwrites the existing source and metadata for the same version", func() {
			vr := Builds.VersionedResource{
				Name:    "some-resource",
				Type:    "some-type",
				Source:  config.Source{"some": "source"},
				Version: Builds.Version{"version": "1"},
				Metadata: []Builds.MetadataField{
					{Name: "meta1", Value: "value1"},
				},
			}

			err := db.SaveVersionedResource(vr)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestVersionedResource("some-resource")).Should(Equal(vr))

			modified := vr
			modified.Source["additional"] = "data"
			modified.Metadata[0].Value = "modified-value1"

			err = db.SaveVersionedResource(modified)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestVersionedResource("some-resource")).Should(Equal(modified))
		})
	})

	Describe("determining the inputs for a job", func() {
		It("ensures that versions from jobs mentioned in two input's 'passed' sections came from the same builds", func() {
			err := db.RegisterJob("job-1")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.RegisterJob("job-2")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.RegisterJob("shared-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.RegisterResource("resource-1")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.RegisterResource("resource-2")
			Ω(err).ShouldNot(HaveOccurred())

			j1b1, err := db.CreateBuild("job-1")
			Ω(err).ShouldNot(HaveOccurred())

			j2b1, err := db.CreateBuild("job-2")
			Ω(err).ShouldNot(HaveOccurred())

			sb1, err := db.CreateBuild("shared-job")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(sb1.ID, Builds.VersionedResource{
				Name:    "resource-1",
				Type:    "some-type",
				Version: Builds.Version{"v": "r1-common-to-shared-and-j1"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(sb1.ID, Builds.VersionedResource{
				Name:    "resource-2",
				Type:    "some-type",
				Version: Builds.Version{"v": "r2-common-to-shared-and-j2"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(j1b1.ID, Builds.VersionedResource{
				Name:    "resource-1",
				Type:    "some-type",
				Version: Builds.Version{"v": "r1-common-to-shared-and-j1"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(j2b1.ID, Builds.VersionedResource{
				Name:    "resource-2",
				Type:    "some-type",
				Version: Builds.Version{"v": "r2-common-to-shared-and-j2"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestInputVersions([]config.Input{
				{
					Resource: "resource-1",
					Passed:   []string{"shared-job", "job-1"},
				},
				{
					Resource: "resource-2",
					Passed:   []string{"shared-job", "job-2"},
				},
			})).Should(Equal(Builds.VersionedResources{
				{
					Name:    "resource-1",
					Type:    "some-type",
					Version: Builds.Version{"v": "r1-common-to-shared-and-j1"},
				},
				{
					Name:    "resource-2",
					Type:    "some-type",
					Version: Builds.Version{"v": "r2-common-to-shared-and-j2"},
				},
			}))

			sb2, err := db.CreateBuild("shared-job")
			Ω(err).ShouldNot(HaveOccurred())

			j1b2, err := db.CreateBuild("job-1")
			Ω(err).ShouldNot(HaveOccurred())

			j2b2, err := db.CreateBuild("job-2")
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(sb2.ID, Builds.VersionedResource{
				Name:    "resource-1",
				Type:    "some-type",
				Version: Builds.Version{"v": "new-r1-common-to-shared-and-j1"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(sb2.ID, Builds.VersionedResource{
				Name:    "resource-2",
				Type:    "some-type",
				Version: Builds.Version{"v": "new-r2-common-to-shared-and-j2"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			err = db.SaveBuildOutput(j1b2.ID, Builds.VersionedResource{
				Name:    "resource-1",
				Type:    "some-type",
				Version: Builds.Version{"v": "new-r1-common-to-shared-and-j1"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			// do NOT save resource-2 as an output of job-2

			Ω(db.GetLatestInputVersions([]config.Input{
				{
					Resource: "resource-1",
					Passed:   []string{"shared-job", "job-1"},
				},
				{
					Resource: "resource-2",
					Passed:   []string{"shared-job", "job-2"},
				},
			})).Should(Equal(Builds.VersionedResources{
				{
					Name:    "resource-1",
					Type:    "some-type",
					Version: Builds.Version{"v": "r1-common-to-shared-and-j1"},
				},
				{
					Name:    "resource-2",
					Type:    "some-type",
					Version: Builds.Version{"v": "r2-common-to-shared-and-j2"},
				},
			}))

			// now save the output of resource-2 job-2
			err = db.SaveBuildOutput(j2b2.ID, Builds.VersionedResource{
				Name:    "resource-2",
				Type:    "some-type",
				Version: Builds.Version{"v": "new-r2-common-to-shared-and-j2"},
			})
			Ω(err).ShouldNot(HaveOccurred())

			Ω(db.GetLatestInputVersions([]config.Input{
				{
					Resource: "resource-1",
					Passed:   []string{"shared-job", "job-1"},
				},
				{
					Resource: "resource-2",
					Passed:   []string{"shared-job", "job-2"},
				},
			})).Should(Equal(Builds.VersionedResources{
				{
					Name:    "resource-1",
					Type:    "some-type",
					Version: Builds.Version{"v": "new-r1-common-to-shared-and-j1"},
				},
				{
					Name:    "resource-2",
					Type:    "some-type",
					Version: Builds.Version{"v": "new-r2-common-to-shared-and-j2"},
				},
			}))

			// save newer versions; should be new latest
			for i := 0; i < 10; i++ {
				version := fmt.Sprintf("version-%d", i+1)

				err = db.SaveBuildOutput(sb1.ID, Builds.VersionedResource{
					Name:    "resource-1",
					Type:    "some-type",
					Version: Builds.Version{"v": version + "-r1-common-to-shared-and-j1"},
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = db.SaveBuildOutput(sb1.ID, Builds.VersionedResource{
					Name:    "resource-2",
					Type:    "some-type",
					Version: Builds.Version{"v": version + "-r2-common-to-shared-and-j2"},
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = db.SaveBuildOutput(j1b1.ID, Builds.VersionedResource{
					Name:    "resource-1",
					Type:    "some-type",
					Version: Builds.Version{"v": version + "-r1-common-to-shared-and-j1"},
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = db.SaveBuildOutput(j2b1.ID, Builds.VersionedResource{
					Name:    "resource-2",
					Type:    "some-type",
					Version: Builds.Version{"v": version + "-r2-common-to-shared-and-j2"},
				})
				Ω(err).ShouldNot(HaveOccurred())

				Ω(db.GetLatestInputVersions([]config.Input{
					{
						Resource: "resource-1",
						Passed:   []string{"shared-job", "job-1"},
					},
					{
						Resource: "resource-2",
						Passed:   []string{"shared-job", "job-2"},
					},
				})).Should(Equal(Builds.VersionedResources{
					{
						Name:    "resource-1",
						Type:    "some-type",
						Version: Builds.Version{"v": version + "-r1-common-to-shared-and-j1"},
					},
					{
						Name:    "resource-2",
						Type:    "some-type",
						Version: Builds.Version{"v": version + "-r2-common-to-shared-and-j2"},
					},
				}))
			}
		})
	})

	Context("when the first build is created", func() {
		var firstBuild Builds.Build

		var job string

		BeforeEach(func() {
			var err error

			job = "some-job"

			firstBuild, err = db.CreateBuild(job)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(firstBuild.Name).Should(Equal("1"))
			Ω(firstBuild.Status).Should(Equal(Builds.StatusPending))
		})

		Context("and then aborted", func() {
			BeforeEach(func() {
				_, err := db.AbortBuild(firstBuild.ID)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("changes the state to aborted", func() {
				build, err := db.GetJobBuild(job, firstBuild.Name)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(build.Status).Should(Equal(Builds.StatusAborted))
			})

			Describe("scheduling the build", func() {
				It("fails", func() {
					scheduled, err := db.ScheduleBuild(job, firstBuild.Name, false)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(scheduled).Should(BeFalse())
				})
			})
		})

		Context("and then scheduled", func() {
			BeforeEach(func() {
				scheduled, err := db.ScheduleBuild(job, firstBuild.Name, false)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(scheduled).Should(BeTrue())
			})

			Context("and then aborted", func() {
				BeforeEach(func() {
					_, err := db.AbortBuild(firstBuild.ID)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("changes the state to aborted", func() {
					build, err := db.GetJobBuild(job, firstBuild.Name)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(build.Status).Should(Equal(Builds.StatusAborted))
				})

				Describe("starting the build", func() {
					It("fails", func() {
						started, err := db.StartBuild(job, firstBuild.Name, "abort-url")
						Ω(err).ShouldNot(HaveOccurred())
						Ω(started).Should(BeFalse())
					})
				})
			})
		})

		Describe("scheduling the build", func() {
			It("succeeds", func() {
				scheduled, err := db.ScheduleBuild(job, firstBuild.Name, false)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(scheduled).Should(BeTrue())
			})

			Context("serially", func() {
				It("succeeds", func() {
					scheduled, err := db.ScheduleBuild(job, firstBuild.Name, true)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(scheduled).Should(BeTrue())
				})
			})
		})

		Context("and second build is created", func() {
			var secondBuild Builds.Build

			BeforeEach(func() {
				var err error

				secondBuild, err = db.CreateBuild(job)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(secondBuild.Name).Should(Equal("2"))
				Ω(secondBuild.Status).Should(Equal(Builds.StatusPending))
			})

			Describe("scheduling the second build", func() {
				It("succeeds", func() {
					scheduled, err := db.ScheduleBuild(job, secondBuild.Name, false)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(scheduled).Should(BeTrue())
				})

				Context("with serial true", func() {
					It("fails", func() {
						scheduled, err := db.ScheduleBuild(job, secondBuild.Name, true)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(scheduled).Should(BeFalse())
					})
				})
			})

			Describe("after the first build schedules", func() {
				BeforeEach(func() {
					scheduled, err := db.ScheduleBuild(job, firstBuild.Name, false)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(scheduled).Should(BeTrue())
				})

				Context("when the second build is scheduled serially", func() {
					It("fails", func() {
						scheduled, err := db.ScheduleBuild(job, secondBuild.Name, true)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(scheduled).Should(BeFalse())
					})
				})

				for _, s := range []Builds.Status{Builds.StatusSucceeded, Builds.StatusFailed, Builds.StatusErrored} {
					status := s

					Context("and the first build's status changes to "+string(status), func() {
						BeforeEach(func() {
							err := db.SaveBuildStatus(firstBuild.ID, status)
							Ω(err).ShouldNot(HaveOccurred())
						})

						Context("and the second build is scheduled serially", func() {
							It("succeeds", func() {
								scheduled, err := db.ScheduleBuild(job, secondBuild.Name, true)
								Ω(err).ShouldNot(HaveOccurred())
								Ω(scheduled).Should(BeTrue())
							})
						})
					})
				}
			})

			Describe("after the first build is aborted", func() {
				BeforeEach(func() {
					_, err := db.AbortBuild(firstBuild.ID)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("when the second build is scheduled serially", func() {
					It("succeeds", func() {
						scheduled, err := db.ScheduleBuild(job, secondBuild.Name, true)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(scheduled).Should(BeTrue())
					})
				})
			})

			Context("and a third build is created", func() {
				var thirdBuild Builds.Build

				BeforeEach(func() {
					var err error

					thirdBuild, err = db.CreateBuild(job)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(thirdBuild.Name).Should(Equal("3"))
					Ω(thirdBuild.Status).Should(Equal(Builds.StatusPending))
				})

				Context("and the first build finishes", func() {
					BeforeEach(func() {
						err := db.SaveBuildStatus(firstBuild.ID, Builds.StatusSucceeded)
						Ω(err).ShouldNot(HaveOccurred())
					})

					Context("and the third build is scheduled serially", func() {
						It("fails, as it would have jumped the queue", func() {
							scheduled, err := db.ScheduleBuild(job, thirdBuild.Name, true)
							Ω(err).ShouldNot(HaveOccurred())
							Ω(scheduled).Should(BeFalse())
						})
					})
				})

				Context("and then scheduled", func() {
					It("succeeds", func() {
						scheduled, err := db.ScheduleBuild(job, thirdBuild.Name, false)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(scheduled).Should(BeTrue())
					})

					Context("with serial true", func() {
						It("fails", func() {
							scheduled, err := db.ScheduleBuild(job, thirdBuild.Name, true)
							Ω(err).ShouldNot(HaveOccurred())
							Ω(scheduled).Should(BeFalse())
						})
					})
				})
			})
		})
	})
}

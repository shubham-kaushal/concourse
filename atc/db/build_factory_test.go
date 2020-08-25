package db_test

import (
	"time"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("BuildFactory", func() {
	var team db.Team

	BeforeEach(func() {
		var err error
		team, err = teamFactory.CreateTeam(atc.Team{Name: "some-team"})
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Build", func() {
		var (
			foundBuild   db.Build
			createdBuild db.Build
			found        bool
		)

		BeforeEach(func() {
			var err error
			createdBuild, err = team.CreateOneOffBuild()
			Expect(err).ToNot(HaveOccurred())

			foundBuild, found, err = buildFactory.Build(createdBuild.ID())
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the correct build", func() {
			Expect(found).To(BeTrue())
			Expect(foundBuild).To(Equal(createdBuild))
		})
	})

	Describe("MarkNonInterceptibleBuilds", func() {
		Context("one-off builds", func() {
			DescribeTable("completed and within grace period",
				func(status db.BuildStatus, matcher types.GomegaMatcher) {
					b, err := defaultTeam.CreateOneOffBuild()
					Expect(err).NotTo(HaveOccurred())

					var i bool
					err = b.Finish(status)
					Expect(err).NotTo(HaveOccurred())

					err = buildFactory.MarkNonInterceptibleBuilds()
					Expect(err).NotTo(HaveOccurred())

					i, err = b.Interceptible()
					Expect(err).NotTo(HaveOccurred())
					Expect(i).To(matcher)
				},
				Entry("succeeded is interceptible", db.BuildStatusSucceeded, BeTrue()),
				Entry("aborted is interceptible", db.BuildStatusAborted, BeTrue()),
				Entry("errored is interceptible", db.BuildStatusErrored, BeTrue()),
				Entry("failed is interceptible", db.BuildStatusFailed, BeTrue()),
			)
			DescribeTable("completed and past the grace period",
				func(status db.BuildStatus, matcher types.GomegaMatcher) {
					//set grace period to 0 for this test
					buildFactory = db.NewBuildFactory(dbConn, lockFactory, 0, 0)
					b, err := defaultTeam.CreateOneOffBuild()
					Expect(err).NotTo(HaveOccurred())

					var i bool
					err = b.Finish(status)
					Expect(err).NotTo(HaveOccurred())

					err = buildFactory.MarkNonInterceptibleBuilds()
					Expect(err).NotTo(HaveOccurred())

					i, err = b.Interceptible()
					Expect(err).NotTo(HaveOccurred())
					Expect(i).To(matcher)
				},
				Entry("succeeded is non-interceptible", db.BuildStatusSucceeded, BeFalse()),
				Entry("aborted is non-interceptible", db.BuildStatusAborted, BeFalse()),
				Entry("errored is non-interceptible", db.BuildStatusErrored, BeFalse()),
				Entry("failed is non-interceptible", db.BuildStatusFailed, BeFalse()),
			)

			It("non-completed is interceptible", func() {
				b, err := defaultTeam.CreateOneOffBuild()
				Expect(err).NotTo(HaveOccurred())

				var i bool
				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())
				i, err = b.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())
			})
		})

		Context("pipeline builds", func() {

			It("[#139963615] marks builds that aren't the latest as non-interceptible, ", func() {
				build1, err := defaultJob.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				build2, err := defaultJob.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				err = build1.Finish(db.BuildStatusErrored)
				Expect(err).NotTo(HaveOccurred())
				err = build2.Finish(db.BuildStatusErrored)
				Expect(err).NotTo(HaveOccurred())

				p, _, err := defaultTeam.SavePipeline("other-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{
							Name: "some-other-job",
						},
					},
				}, db.ConfigVersion(0), false)
				Expect(err).NotTo(HaveOccurred())

				j, found, err := p.Job("some-other-job")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())

				pb1, err := j.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				pb2, err := j.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				err = pb1.Finish(db.BuildStatusErrored)
				Expect(err).NotTo(HaveOccurred())
				err = pb2.Finish(db.BuildStatusErrored)
				Expect(err).NotTo(HaveOccurred())

				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())

				var i bool
				i, err = build1.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeFalse())

				i, err = build2.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())

				i, err = pb1.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeFalse())

				i, err = pb2.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())

			})

			DescribeTable("completed builds",
				func(status db.BuildStatus, matcher types.GomegaMatcher) {
					b, err := defaultJob.CreateBuild()
					Expect(err).NotTo(HaveOccurred())

					var i bool

					err = b.Finish(status)
					Expect(err).NotTo(HaveOccurred())

					err = buildFactory.MarkNonInterceptibleBuilds()
					Expect(err).NotTo(HaveOccurred())
					i, err = b.Interceptible()
					Expect(err).NotTo(HaveOccurred())
					Expect(i).To(matcher)
				},
				Entry("succeeded is non-interceptible", db.BuildStatusSucceeded, BeFalse()),
				Entry("aborted is interceptible", db.BuildStatusAborted, BeTrue()),
				Entry("errored is interceptible", db.BuildStatusErrored, BeTrue()),
				Entry("failed is interceptible", db.BuildStatusFailed, BeTrue()),
			)

			It("does not mark non-completed builds", func() {
				b, err := defaultJob.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				var i bool
				i, err = b.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())

				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())
				i, err = b.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())

				_, err = b.Start(atc.Plan{})
				Expect(err).NotTo(HaveOccurred())

				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())
				i, err = b.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())
			})
		})
		Context("GC failed builds", func() {
			It("marks failed builds non-interceptible after failed-grace-period", func() {
				buildFactory = db.NewBuildFactory(dbConn, lockFactory, 0, 2*time.Second) // 1 second could create a flaky test
				build, err := defaultJob.CreateBuild()
				Expect(err).NotTo(HaveOccurred())

				err = build.Finish(db.BuildStatusFailed)
				Expect(err).NotTo(HaveOccurred())

				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())

				var i bool
				i, err = build.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeTrue())

				time.Sleep(3 * time.Second) // Wait is too long, only second granularity, better method?

				err = buildFactory.MarkNonInterceptibleBuilds()
				Expect(err).NotTo(HaveOccurred())

				i, err = build.Interceptible()
				Expect(err).NotTo(HaveOccurred())
				Expect(i).To(BeFalse())
			})
		})
	})

	Describe("VisibleBuilds", func() {
		var err error
		var build1 db.Build
		var build2 db.Build
		var build3 db.Build
		var build4 db.Build
		var build5 db.Build

		BeforeEach(func() {
			build1, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			config := atc.Config{Jobs: atc.JobConfigs{{Name: "some-job"}}}
			privatePipeline, _, err := team.SavePipeline("private-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())

			privateJob, found, err := privatePipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build2, err = privateJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			publicPipeline, _, err := team.SavePipeline("public-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())
			err = publicPipeline.Expose()
			Expect(err).NotTo(HaveOccurred())

			publicJob, found, err := publicPipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build3, err = publicJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			otherTeam, err := teamFactory.CreateTeam(atc.Team{Name: "some-other-team"})
			Expect(err).NotTo(HaveOccurred())

			build4, err = otherTeam.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			build5, err = privateJob.RerunBuild(build2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns visible builds for the given teams", func() {
			builds, _, err := buildFactory.VisibleBuilds([]string{"some-team"}, db.Page{Limit: 10})
			Expect(err).NotTo(HaveOccurred())

			Expect(builds).To(HaveLen(4))

			buildIDs := []int{}
			for _, build := range builds {
				buildIDs = append(buildIDs, build.ID())
			}
			Expect(buildIDs).To(Equal([]int{build3.ID(), build5.ID(), build2.ID(), build1.ID()}))
			Expect(builds).NotTo(ContainElement(build4))
		})
	})

	Describe("AllBuilds", func() {
		var err error
		var build1 db.Build
		var build2 db.Build
		var build3 db.Build
		var build4 db.Build

		BeforeEach(func() {
			build1, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			config := atc.Config{Jobs: atc.JobConfigs{{Name: "some-job"}}}
			privatePipeline, _, err := team.SavePipeline("private-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())

			privateJob, found, err := privatePipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build2, err = privateJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			publicPipeline, _, err := team.SavePipeline("public-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())
			err = publicPipeline.Expose()
			Expect(err).NotTo(HaveOccurred())

			publicJob, found, err := publicPipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build3, err = publicJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			otherTeam, err := teamFactory.CreateTeam(atc.Team{Name: "some-other-team"})
			Expect(err).NotTo(HaveOccurred())

			build4, err = otherTeam.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all builds from all teams private and public pipelines", func() {
			builds, _, err := buildFactory.AllBuilds(db.Page{Limit: 10})
			Expect(err).NotTo(HaveOccurred())

			Expect(builds).To(HaveLen(4))
			Expect(builds).To(ConsistOf(build1, build2, build3, build4))
		})
	})

	Describe("PublicBuilds", func() {
		var publicBuild db.Build

		BeforeEach(func() {
			_, err := team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			config := atc.Config{Jobs: atc.JobConfigs{{Name: "some-job"}}}
			privatePipeline, _, err := team.SavePipeline("private-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())

			privateJob, found, err := privatePipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			_, err = privateJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			publicPipeline, _, err := team.SavePipeline("public-pipeline", config, db.ConfigVersion(1), false)
			Expect(err).NotTo(HaveOccurred())
			err = publicPipeline.Expose()
			Expect(err).NotTo(HaveOccurred())

			publicJob, found, err := publicPipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			publicBuild, err = publicJob.CreateBuild()
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns public builds", func() {
			builds, _, err := buildFactory.PublicBuilds(db.Page{Limit: 10})
			Expect(err).NotTo(HaveOccurred())

			Expect(builds).To(HaveLen(1))
			Expect(builds).To(ConsistOf(publicBuild))
		})
	})

	Describe("GetDrainableBuilds", func() {
		var build2DB, build3DB, build4DB db.Build

		BeforeEach(func() {
			pipeline, _, err := team.SavePipeline("other-pipeline", atc.Config{
				Jobs: atc.JobConfigs{
					{
						Name: "some-job",
					},
				},
			}, db.ConfigVersion(0), false)
			Expect(err).NotTo(HaveOccurred())

			job, found, err := pipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			_, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			build2DB, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			build3DB, err = job.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			build4DB, err = job.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			started, err := build2DB.Start(atc.Plan{})
			Expect(err).NotTo(HaveOccurred())
			Expect(started).To(BeTrue())

			err = build3DB.Finish("succeeded")
			Expect(err).NotTo(HaveOccurred())

			err = build3DB.SetDrained(true)
			Expect(err).NotTo(HaveOccurred())

			err = build4DB.Finish("failed")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all builds that have been completed and not drained", func() {
			builds, err := buildFactory.GetDrainableBuilds()
			Expect(err).NotTo(HaveOccurred())

			_, err = build4DB.Reload()
			Expect(err).NotTo(HaveOccurred())

			Expect(builds).To(ConsistOf(build4DB))
		})
	})

	Describe("GetAllStartedBuilds", func() {
		var build1DB db.Build
		var build2DB db.Build

		BeforeEach(func() {
			pipeline, _, err := team.SavePipeline("other-pipeline", atc.Config{
				Jobs: atc.JobConfigs{
					{
						Name: "some-job",
					},
				},
			}, db.ConfigVersion(0), false)
			Expect(err).NotTo(HaveOccurred())

			job, found, err := pipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build1DB, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			build2DB, err = job.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			_, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			started, err := build1DB.Start(atc.Plan{})
			Expect(err).NotTo(HaveOccurred())
			Expect(started).To(BeTrue())

			started, err = build2DB.Start(atc.Plan{})
			Expect(err).NotTo(HaveOccurred())
			Expect(started).To(BeTrue())
		})

		It("returns all builds that have been started, regardless of pipeline", func() {
			builds, err := buildFactory.GetAllStartedBuilds()
			Expect(err).NotTo(HaveOccurred())

			_, err = build1DB.Reload()
			Expect(err).NotTo(HaveOccurred())
			_, err = build2DB.Reload()
			Expect(err).NotTo(HaveOccurred())

			Expect(builds).To(ConsistOf(build1DB, build2DB))
		})
	})

	Describe("AllBuilds by date", func() {
		var build1DB db.Build
		var build2DB db.Build

		BeforeEach(func() {
			pipeline, _, err := team.SavePipeline("other-pipeline", atc.Config{
				Jobs: atc.JobConfigs{
					{
						Name: "some-job",
					},
				},
			}, db.ConfigVersion(0), false)
			Expect(err).NotTo(HaveOccurred())

			job, found, err := pipeline.Job("some-job")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			build1DB, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			build2DB, err = job.CreateBuild()
			Expect(err).NotTo(HaveOccurred())

			_, err = team.CreateOneOffBuild()
			Expect(err).NotTo(HaveOccurred())

			started, err := build1DB.Start(atc.Plan{})
			Expect(err).NotTo(HaveOccurred())
			Expect(started).To(BeTrue())

			started, err = build2DB.Start(atc.Plan{})
			Expect(err).NotTo(HaveOccurred())
			Expect(started).To(BeTrue())
		})

		It("returns builds in the given timespan", func() {
			page := db.Page{
				Limit:   10,
				From:    int(time.Now().Unix() - 10000),
				To:      int(time.Now().Unix() + 10),
				UseDate: true,
			}
			builds, _, err := buildFactory.AllBuilds(page)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(builds)).To(Equal(3))
		})

		Describe("with a future date as Page.From", func() {
			It("should return nothing", func() {
				page := db.Page{
					Limit:   10,
					From:    int(time.Now().Unix() + 10),
					UseDate: true,
				}
				builds, _, err := buildFactory.AllBuilds(page)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(builds)).To(Equal(0))
			})
		})

		Describe("with a very old date as Page.To", func() {
			It("should return nothing", func() {
				page := db.Page{
					Limit:   10,
					To:      int(time.Now().Unix() - 10000),
					UseDate: true,
				}
				builds, _, err := buildFactory.AllBuilds(page)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(builds)).To(Equal(0))
			})
		})
	})
})

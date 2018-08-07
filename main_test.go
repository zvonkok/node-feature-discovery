package main

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/kubernetes-incubator/node-feature-discovery/source"
	"github.com/kubernetes-incubator/node-feature-discovery/source/fake"
	"github.com/kubernetes-incubator/node-feature-discovery/source/panic_fake"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/vektra/errors"
	api "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

func TestDiscoveryWithMockSources(t *testing.T) {
	Convey("When I discover features from fake source and update the node using fake client", t, func() {
		mockFeatureSource := new(MockFeatureSource)
		fakeFeatureSourceName := string("testSource")
		fakeFeatures := []string{"testfeature1", "testfeature2", "testfeature3"}
		fakeFeatureLabels := Labels{}
		for _, f := range fakeFeatures {
			fakeFeatureLabels[fmt.Sprintf("%s-testSource-%s", prefix, f)] = "true"
		}
		fakeFeatureSource := source.FeatureSource(mockFeatureSource)

		Convey("When I successfully get the labels from the mock source", func() {
			mockFeatureSource.On("Name").Return(fakeFeatureSourceName)
			mockFeatureSource.On("Discover").Return(fakeFeatures, nil)

			returnedLabels, err := getFeatureLabels(fakeFeatureSource)
			Convey("Proper label is returned", func() {
				So(returnedLabels, ShouldResemble, fakeFeatureLabels)
			})
			Convey("Error is nil", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When I fail to get the labels from the mock source", func() {
			expectedError := errors.New("fake error")
			mockFeatureSource.On("Discover").Return(nil, expectedError)

			returnedLabels, err := getFeatureLabels(fakeFeatureSource)
			Convey("No label is returned", func() {
				So(returnedLabels, ShouldBeNil)
			})
			Convey("Error is produced", func() {
				So(err, ShouldEqual, expectedError)
			})
		})

		mockAPIHelper := new(MockAPIHelpers)
		testHelper := APIHelpers(mockAPIHelper)
		var mockClient *k8sclient.Clientset
		var mockNode *api.Node

		Convey("When I successfully update the node with feature labels", func() {
			mockAPIHelper.On("GetClient").Return(mockClient, nil)
			mockAPIHelper.On("GetNode", mockClient).Return(mockNode, nil).Once()
			mockAPIHelper.On("AddLabels", mockNode, fakeFeatureLabels).Return().Once()
			mockAPIHelper.On("RemoveLabels", mockNode, prefix).Return().Once()
			mockAPIHelper.On("UpdateNode", mockClient, mockNode).Return(nil).Once()
			noPublish := false
			err := updateNodeWithFeatureLabels(testHelper, noPublish, fakeFeatureLabels)

			Convey("Error is nil", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When I fail to update the node with feature labels", func() {
			expectedError := errors.New("fake error")
			mockAPIHelper.On("GetClient").Return(nil, expectedError)
			noPublish := false
			err := updateNodeWithFeatureLabels(testHelper, noPublish, fakeFeatureLabels)

			Convey("Error is produced", func() {
				So(err, ShouldEqual, expectedError)
			})
		})

		Convey("When I fail to get a mock client while advertising feature labels", func() {
			expectedError := errors.New("fake error")
			mockAPIHelper.On("GetClient").Return(nil, expectedError)
			err := advertiseFeatureLabels(testHelper, fakeFeatureLabels)

			Convey("Error is produced", func() {
				So(err, ShouldEqual, expectedError)
			})
		})

		Convey("When I fail to get a mock node while advertising feature labels", func() {
			expectedError := errors.New("fake error")
			mockAPIHelper.On("GetClient").Return(mockClient, nil)
			mockAPIHelper.On("GetNode", mockClient).Return(nil, expectedError).Once()
			err := advertiseFeatureLabels(testHelper, fakeFeatureLabels)

			Convey("Error is produced", func() {
				So(err, ShouldEqual, expectedError)
			})
		})

		Convey("When I fail to update a mock node while advertising feature labels", func() {
			expectedError := errors.New("fake error")
			mockAPIHelper.On("GetClient").Return(mockClient, nil)
			mockAPIHelper.On("GetNode", mockClient).Return(mockNode, nil).Once()
			mockAPIHelper.On("RemoveLabels", mockNode, prefix).Return().Once()
			mockAPIHelper.On("AddLabels", mockNode, fakeFeatureLabels).Return().Once()
			mockAPIHelper.On("UpdateNode", mockClient, mockNode).Return(expectedError).Once()
			err := advertiseFeatureLabels(testHelper, fakeFeatureLabels)

			Convey("Error is produced", func() {
				So(err, ShouldEqual, expectedError)
			})
		})

	})
}

func TestArgsParse(t *testing.T) {
	Convey("When parsing command line arguments", t, func() {
		argv1 := []string{"--no-publish", "--oneshot"}
		argv2 := []string{"--sources=fake1,fake2,fake3", "--sleep-interval=30s"}
		argv3 := []string{"--label-whitelist=.*rdt.*"}
		argv4 := []string{"--no-publish", "--sources=fake1,fake2,fake3"}

		Convey("When --no-publish and --oneshot flags are passed", func() {
			args := argsParse(argv1)

			Convey("noPublish is set and args.sources is set to the default value", func() {
				So(args.sleepInterval, ShouldEqual, 60*time.Second)
				So(args.noPublish, ShouldBeTrue)
				So(args.oneshot, ShouldBeTrue)
				So(args.sources, ShouldResemble, []string{"cpuid", "iommu", "memory", "network", "pstate", "rdt", "selinux", "storage", "gpu"})
				So(len(args.labelWhiteList), ShouldEqual, 0)
			})
		})

		Convey("When --sources flag is passed and set to some values, --sleep-inteval is specified", func() {
			args := argsParse(argv2)

			Convey("args.sources is set to appropriate values", func() {
				So(args.sleepInterval, ShouldEqual, 30*time.Second)
				So(args.noPublish, ShouldBeFalse)
				So(args.oneshot, ShouldBeFalse)
				So(args.sources, ShouldResemble, []string{"fake1", "fake2", "fake3"})
				So(len(args.labelWhiteList), ShouldEqual, 0)
			})
		})

		Convey("When --label-whitelist flag is passed and set to some value", func() {
			args := argsParse(argv3)

			Convey("args.labelWhiteList is set to appropriate value and args.sources is set to default value", func() {
				So(args.noPublish, ShouldBeFalse)
				So(args.sources, ShouldResemble, []string{"cpuid", "iommu", "memory", "network", "pstate", "rdt", "selinux", "storage"})
				So(args.labelWhiteList, ShouldResemble, ".*rdt.*")
			})
		})

		Convey("When --no-publish and --sources flag are passed and --sources flag is set to some value", func() {
			args := argsParse(argv4)

			Convey("--no-publish is set and args.sources is set to appropriate values", func() {
				So(args.noPublish, ShouldBeTrue)
				So(args.sources, ShouldResemble, []string{"fake1", "fake2", "fake3"})
				So(len(args.labelWhiteList), ShouldEqual, 0)
			})
		})
	})
}

func TestConfigureParameters(t *testing.T) {
	Convey("When configuring parameters for node feature discovery", t, func() {

		Convey("When no sourcesWhiteList and labelWhiteListStr are passed", func() {
			sourcesWhiteList := []string{}
			labelWhiteListStr := ""
			emptyRegexp, _ := regexp.Compile("")
			enabledSources, labelWhiteList, err := configureParameters(sourcesWhiteList, labelWhiteListStr)

			Convey("Error should not be produced", func() {
				So(err, ShouldBeNil)
			})
			Convey("No sourcesWhiteList or labelWhiteList are returned", func() {
				So(len(enabledSources), ShouldEqual, 0)
				So(labelWhiteList, ShouldResemble, emptyRegexp)
			})
		})

		Convey("When sourcesWhiteList is passed", func() {
			sourcesWhiteList := []string{"fake"}
			labelWhiteListStr := ""
			emptyRegexp, _ := regexp.Compile("")
			enabledSources, labelWhiteList, err := configureParameters(sourcesWhiteList, labelWhiteListStr)

			Convey("Error should not be produced", func() {
				So(err, ShouldBeNil)
			})
			Convey("Proper sourcesWhiteList are returned", func() {
				So(len(enabledSources), ShouldEqual, 1)
				So(enabledSources[0], ShouldHaveSameTypeAs, fake.Source{})
				So(labelWhiteList, ShouldResemble, emptyRegexp)
			})
		})

		Convey("When invalid labelWhiteListStr is passed", func() {
			sourcesWhiteList := []string{""}
			labelWhiteListStr := "*"
			enabledSources, labelWhiteList, err := configureParameters(sourcesWhiteList, labelWhiteListStr)

			Convey("Error is produced", func() {
				So(enabledSources, ShouldBeNil)
				So(labelWhiteList, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When valid labelWhiteListStr is passed", func() {
			sourcesWhiteList := []string{""}
			labelWhiteListStr := ".*rdt.*"
			expectRegexp, err := regexp.Compile(".*rdt.*")
			enabledSources, labelWhiteList, err := configureParameters(sourcesWhiteList, labelWhiteListStr)

			Convey("Error should not be produced", func() {
				So(err, ShouldBeNil)
			})
			Convey("Proper labelWhiteList is returned", func() {
				So(len(enabledSources), ShouldEqual, 0)
				So(labelWhiteList, ShouldResemble, expectRegexp)
			})
		})
	})
}

func TestCreateFeatureLabels(t *testing.T) {
	Convey("When creating feature labels from the configured sources", t, func() {
		Convey("When fake feature source is configured", func() {
			emptyLabelWL, _ := regexp.Compile("")
			fakeFeatureSource := source.FeatureSource(new(fake.Source))
			sources := []source.FeatureSource{}
			sources = append(sources, fakeFeatureSource)
			labels := createFeatureLabels(sources, emptyLabelWL)

			Convey("Proper fake labels are returned", func() {
				So(len(labels), ShouldEqual, 4)
				So(labels, ShouldContainKey, prefix+"-fake-fakefeature1")
				So(labels, ShouldContainKey, prefix+"-fake-fakefeature2")
				So(labels, ShouldContainKey, prefix+"-fake-fakefeature3")
			})
		})
		Convey("When fake feature source is configured with a whitelist that doesn't match", func() {
			emptyLabelWL, _ := regexp.Compile(".*rdt.*")
			fakeFeatureSource := source.FeatureSource(new(fake.Source))
			sources := []source.FeatureSource{}
			sources = append(sources, fakeFeatureSource)
			labels := createFeatureLabels(sources, emptyLabelWL)

			Convey("fake labels are not returned", func() {
				So(len(labels), ShouldEqual, 1)
				So(labels, ShouldNotContainKey, prefix+"-fake-fakefeature1")
				So(labels, ShouldNotContainKey, prefix+"-fake-fakefeature2")
				So(labels, ShouldNotContainKey, prefix+"-fake-fakefeature3")
			})
		})
	})
}

func TestAddLabels(t *testing.T) {
	Convey("When adding labels", t, func() {
		helper := k8sHelpers{}
		labels := Labels{}
		n := &api.Node{
			ObjectMeta: meta_v1.ObjectMeta{
				Labels: map[string]string{},
			},
		}

		Convey("If no labels are passed", func() {
			helper.AddLabels(n, labels)

			Convey("None should be added", func() {
				So(len(n.Labels), ShouldEqual, 0)
			})
		})

		Convey("They should be added to the node.Labels", func() {
			test1 := prefix + ".test1"
			labels[test1] = "true"
			helper.AddLabels(n, labels)
			So(n.Labels, ShouldContainKey, test1)
		})
	})
}

func TestRemoveLabels(t *testing.T) {
	Convey("When removing labels", t, func() {
		helper := k8sHelpers{}
		n := &api.Node{
			ObjectMeta: meta_v1.ObjectMeta{
				Labels: map[string]string{
					"single":     "123",
					"multiple_A": "a",
					"multiple_B": "b",
				},
			},
		}

		Convey("a unique label should be removed", func() {
			helper.RemoveLabels(n, "single")
			So(len(n.Labels), ShouldEqual, 2)
			So(n.Labels, ShouldNotContainKey, "single")
		})

		Convey("a non-unique search string should remove all matching keys", func() {
			helper.RemoveLabels(n, "multiple")
			So(len(n.Labels), ShouldEqual, 1)
			So(n.Labels, ShouldNotContainKey, "multiple_A")
			So(n.Labels, ShouldNotContainKey, "multiple_B")
		})

		Convey("a search string with no matches should not alter labels", func() {
			helper.RemoveLabels(n, "unique")
			So(n.Labels, ShouldContainKey, "single")
			So(n.Labels, ShouldContainKey, "multiple_A")
			So(n.Labels, ShouldContainKey, "multiple_B")
			So(len(n.Labels), ShouldEqual, 3)
		})
	})
}

func TestGetFeatureLabels(t *testing.T) {
	Convey("When I get feature labels and panic occurs during discovery of a feature source", t, func() {
		fakePanicFeatureSource := source.FeatureSource(new(panic_fake.Source))

		returnedLabels, err := getFeatureLabels(fakePanicFeatureSource)
		Convey("No label is returned", func() {
			So(len(returnedLabels), ShouldEqual, 0)
		})
		Convey("Error is produced and panic error is returned", func() {
			So(err, ShouldResemble, fmt.Errorf("fake panic error"))
		})

	})
}

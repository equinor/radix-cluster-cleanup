package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/equinor/radix-acr-cleanup/pkg/delaytick"
	"github.com/equinor/radix-acr-cleanup/pkg/timewindow"
	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"

	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
)

const defaultInactiveDaysBeforeDeletion = 7 * 4
const defaultInactiveDaysBeforeStop = 7
const defaultLogLevel = "INFO"

var rootLongHelp = strings.TrimSpace(`
	A command line interface which allows you to list and automatically delete inactive RadixRegistrations.
`)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rx-cleanup",
	Short: "Command line interface for cleaning up inactive RadixRegistrations",
	Long:  rootLongHelp,
}

// Execute the top level command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("parsing command line arguments failed:", err)
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Int64(settings.InactiveDaysBeforeDeletionOption, defaultInactiveDaysBeforeDeletion, "max inactivity period before deleting RadixRegistrations")
	rootCmd.PersistentFlags().Int64(settings.InactiveDaysBeforeStopOption, defaultInactiveDaysBeforeStop, "max inactivity period before stopping components in RadixRegistrations")
	rootCmd.PersistentFlags().String(settings.WhitelistOption, "", "custom whitelist of RadixRegistrations to exclude from cleanup. Appended to default, hardcoded whitelist")
	rootCmd.PersistentFlags().StringSlice(settings.CleanUpDaysOption, []string{"mo", "tu", "we", "th", "fr", "sa", "su"}, "for commands that run continuously, this option specifies which weekdays the command will be active")
	rootCmd.PersistentFlags().String(settings.CleanUpStartOption, "06:00", "for commands that run continuously, this option specifies which time of day the command will be active from")
	rootCmd.PersistentFlags().String(settings.CleanUpEndOption, "09:00", "for commands that run continuously, this option specifies which time of day the command will be active to")
	rootCmd.PersistentFlags().Duration(settings.CleanUpPeriodOption, time.Minute*30, "for commands that run continuously, this option specifies how long between each consecutive run of the command")
	logLevel, logErr := os.Getenv("LOG_LEVEL"), error(nil)
	if logErr != nil {
		logLevel = defaultLogLevel
	}
	switch logLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func getWhitelist() []string {
	hardcodedWhitelist := []string{
		"radix-api",
		"radix-platform",
		"radix-web-console",
		"radix-vulnerability-scanner",
		"radix-github-webhook",
		"radix-canary-golang",
		"radix-vulnerability-scanner-api",
		"radix-servicenow-proxy",
		"radix-networkpolicy-canary",
		"radix-cost-allocation-api",
		"canarycicd-test1",
		"canarycicd-test2",
		"canarycicd-test3",
		"canarycicd-test4",
	}
	argWhitelist, _ := rootCmd.Flags().GetString(settings.WhitelistOption)
	whitelist := append(hardcodedWhitelist, strings.Split(argWhitelist, ",")...)
	return whitelist
}

func getKubernetesClient() (kubernetes.Interface, radixclient.Interface) {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("getClusterConfig InClusterConfig: %v", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("getClusterConfig k8s client: %v", err)
	}

	radixClient, err := radixclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("getClusterConfig radix client: %v", err)
	}

	log.Printf("Successfully constructed k8s client to API server %v", config.Host)
	return client, radixClient
}

func getKubeUtil() (*kube.Kube, error) {
	kubeClient, radixClient := getKubernetesClient()
	kubeutil, err := kube.New(kubeClient, radixClient, nil)
	if err != nil {
		return nil, err
	}
	return kubeutil, nil
}

func runFunctionPeriodically(someFunc func() error) error {
	cleanupDays, cleanupDaysErr := rootCmd.Flags().GetStringSlice(settings.CleanUpDaysOption)
	cleanupStart, cleanupStartErr := rootCmd.Flags().GetString(settings.CleanUpStartOption)
	cleanupEnd, cleanupEndErr := rootCmd.Flags().GetString(settings.CleanUpEndOption)
	period, periodErr := rootCmd.Flags().GetDuration(settings.CleanUpPeriodOption)
	err := commonErrors.Concat([]error{cleanupDaysErr, cleanupStartErr, cleanupEndErr, periodErr})
	if err != nil {
		return err
	}
	timezone := "Local"
	window, err := timewindow.New(cleanupDays, cleanupStart, cleanupEnd, timezone)
	if err != nil {
		log.Fatalf("Failed to build time window: %v", err)
	}
	source := rand.NewSource(time.Now().UnixNano())
	tick := delaytick.New(source, period)
	for range tick {
		pointInTime := time.Now()
		if window.Contains(pointInTime) {
			log.Infof("Start listing RRs for stop %s", pointInTime)
			err := someFunc()
			if err != nil {
				return err
			}
		} else {
			log.Infof("%s is outside of window. Continue sleeping", pointInTime)
		}
	}
	log.Warnf("execution reached code which was presumably after an inescapable loop")
	return nil
}

func getTooInactiveRrs(kubeClient *kube.Kube, inactivityLimit time.Duration, action string) ([]v1.RadixRegistration, error) {
	rrs, err := kubeClient.ListRegistrations()
	if err != nil {
		return nil, err
	}
	var rrsForDeletion []v1.RadixRegistration
	for _, rr := range rrs {
		if isWhitelisted(rr) {
			log.Debugf("RadixRegistration %s is whitelisted, skipping", rr.Name)
			continue
		}
		ra, err := getRadixApplication(kubeClient, rr.Name)
		if errors.IsNotFound(err) {
			log.Debugf("could not find RadixApplication %s, continuing...", rr.Name)
			continue
		}
		if err != nil {
			return nil, err
		}
		namespaces := getRuntimeNamespaces(ra)
		log.Debugf("found namespaces %s associated with RadixRegistration %s", strings.Join(namespaces, ", "), rr.Name)
		rdsForRr, err := getRadixDeploymentsInNamespaces(kubeClient, namespaces)
		if err != nil {
			return nil, err
		}
		rjsForRr, err := getRadixJobsInNamespace(kubeClient, utils.GetAppNamespace(rr.Name))
		if err != nil {
			return nil, err
		}
		log.Debugf("RadixRegistration %s has %d RadixDeployments, checking them for inactivity", rr.Name, len(rdsForRr))
		isInactive, err := rrIsInactive(rdsForRr, rjsForRr, inactivityLimit, action)
		if err != nil {
			return nil, err
		}
		if isInactive {
			rrsForDeletion = append(rrsForDeletion, *rr)
		}
	}
	return rrsForDeletion, nil
}

func getRadixJobsInNamespace(kubeClient *kube.Kube, namespace string) ([]v1.RadixJob, error) {
	rjs, err := kubeClient.RadixClient().RadixV1().RadixJobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rjs.Items, nil
}

func getRadixDeploymentsInNamespaces(kubeClient *kube.Kube, namespaces []string) ([]v1.RadixDeployment, error) {
	rdsForRr := make([]v1.RadixDeployment, 0)
	for _, ns := range namespaces {
		rds, err := kubeClient.RadixClient().RadixV1().RadixDeployments(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		rdsForRr = append(rdsForRr, rds.Items...)
	}
	return rdsForRr, nil
}

func getRuntimeNamespaces(ra *v1.RadixApplication) []string {
	namespaces := make([]string, 0)
	for _, env := range ra.Spec.Environments {
		namespaces = append(namespaces, utils.GetEnvironmentNamespace(ra.Name, env.Name))
	}
	return namespaces
}

func getRadixApplication(kubeClient *kube.Kube, appName string) (*v1.RadixApplication, error) {
	return kubeClient.RadixClient().RadixV1().RadixApplications(utils.GetAppNamespace(appName)).Get(context.TODO(), appName, metav1.GetOptions{})
}

func isWhitelisted(rr *v1.RadixRegistration) bool {
	whitelist := getWhitelist()
	for _, item := range whitelist {
		if rr.Name == item {
			return true
		}
	}
	return false
}

func rrIsInactive(rds []v1.RadixDeployment, rjs []v1.RadixJob, inactivityLimit time.Duration, action string) (bool, error) {
	if len(rds) == 0 {
		log.Debugf("no RadixDeployments found, assuming RadixRegistration is inactive")
		return true, nil
	}
	latestRadixDeployment := SortDeploymentsByActiveFromTimestampAsc(rds)[len(rds)-1]
	latestRadixDeploymentTimestamp := latestRadixDeployment.Status.ActiveFrom
	latestRadixJobTimestamp := metav1.Time{Time: time.Unix(0, 0)}
	latestRadixJob := getLatestRadixJob(rjs)
	if latestRadixJob != nil {
		latestRadixJobTimestamp = *latestRadixJob.Status.Created
	}

	log.Debugf("most recent radixDeployment is %s, active from %s, %d hours ago", latestRadixDeployment.Name, latestRadixDeploymentTimestamp.Format(time.RFC822), int(time.Since(latestRadixDeploymentTimestamp.Time).Hours()))
	log.Debugf("most recent radixJob was %s, created %s, %d hours ago", latestRadixJob.Name, latestRadixJobTimestamp.Format(time.RFC822), int(time.Since(latestRadixJobTimestamp.Time).Hours()))
	latestUserMutationTimestamp, err := getLastUserMutationTimestamp(latestRadixDeployment)
	if err != nil {
		return false, err
	}
	log.Debugf("most recent manual user activity was %s, %d hours ago", latestUserMutationTimestamp.Format(time.RFC822), int(time.Since(latestUserMutationTimestamp.Time).Hours()))
	lastActivity := getMostRecentTimestamp(&latestRadixJobTimestamp, latestUserMutationTimestamp, &latestRadixDeploymentTimestamp)
	if tooLongInactivity(lastActivity, inactivityLimit) {
		log.Infof("%s: last activity was %d hours ago, which is more than %d hours ago, marking for %s", latestRadixDeployment.Spec.AppName, int(time.Since(lastActivity.Time).Hours()), int(inactivityLimit.Hours()), action)
		return true, nil
	}
	return false, nil
}

func getLatestRadixJob(rjs []v1.RadixJob) *v1.RadixJob {
	if len(rjs) > 0 {
		return &SortJobsByTimestampAsc(rjs)[len(rjs)-1]
	}
	return nil
}

func getLastUserMutationTimestamp(radixDeployment v1.RadixDeployment) (*metav1.Time, error) {
	latestUserMutationTimestamp := metav1.Time{Time: time.Unix(0, 0)}
	latestUserMutation, ok := radixDeployment.Annotations["radix.equinor.com/last-user-mutation"]
	if ok {
		timestamp, err := time.Parse(time.RFC3339, latestUserMutation)
		if err != nil {
			return nil, err
		}
		latestUserMutationTimestamp = metav1.Time{
			Time: timestamp,
		}
	}
	return &latestUserMutationTimestamp, nil
}

func getMostRecentTimestamp(timestamps ...*metav1.Time) *metav1.Time {
	highestTimestamp := &metav1.Time{Time: time.Unix(0, 0)}
	for _, timestamp := range timestamps {
		if timestamp.After(highestTimestamp.Time) {
			highestTimestamp = timestamp
		}
	}
	return highestTimestamp
}

func tooLongInactivity(lastActivity *metav1.Time, ageLimit time.Duration) bool {
	return lastActivity.Unix() < time.Now().Add(-ageLimit).Unix()
}

func SortJobsByTimestampAsc(rjs []v1.RadixJob) []v1.RadixJob {
	sort.Slice(rjs, func(i, j int) bool {
		return isRJ1CreatedAfterRJ2(&rjs[i], &rjs[j])
	})
	return rjs
}

func isRJ1CreatedAfterRJ2(rj1 *v1.RadixJob, rj2 *v1.RadixJob) bool {
	rj1Created := rj1.CreationTimestamp
	rj2Created := rj2.CreationTimestamp
	return rj1Created.Before(&rj2Created)
}

func SortDeploymentsByActiveFromTimestampAsc(rds []v1.RadixDeployment) []v1.RadixDeployment {
	sort.Slice(rds, func(i, j int) bool {
		return isRD1ActiveAfterRD2(&rds[j], &rds[i])
	})
	return rds
}

func isRD1ActiveAfterRD2(rd1 *v1.RadixDeployment, rd2 *v1.RadixDeployment) bool {
	rj1ActiveFrom := rd1.Status.ActiveFrom
	rj2ActiveFrom := rd2.Status.ActiveFrom
	return rj2ActiveFrom.Before(&rj1ActiveFrom)
}

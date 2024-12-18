package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"github.com/equinor/radix-common/utils/delaytick"
	"github.com/equinor/radix-common/utils/timewindow"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
)

const defaultInactiveDaysBeforeDeletion = 7 * 4
const defaultInactiveDaysBeforeStop = 7

var rootLongHelp = strings.TrimSpace(`
	A command line interface which allows you to list and automatically delete inactive RadixRegistrations.
`)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rx-cleanup",
	Short: "Command line interface for cleaning up inactive RadixRegistrations",
	Long:  rootLongHelp,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logLevel, err := cmd.Flags().GetString(settings.LogLevel)
		if err != nil {
			return err
		}

		prettyPrint, err := cmd.Flags().GetBool(settings.PrettyPrint)
		if err != nil {
			return err
		}

		return initZeroLogger(logLevel, prettyPrint)
	},
}

// Execute the top level command
func Execute(ctx context.Context) {
	rootCmd.SetContext(ctx)
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

	rootCmd.PersistentFlags().Bool(settings.PrettyPrint, false, "Enable colored log output instead of json")
	rootCmd.PersistentFlags().String(settings.LogLevel, "info", "Set output log level, allowed values: debug, info, warn, error or fatal")
}

func initZeroLogger(logLevel string, prettyPrint bool) error {
	if logLevel == "" {
		logLevel = zerolog.InfoLevel.String()
	}

	zerologLevel, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(zerologLevel)
	zerolog.DurationFieldUnit = time.Millisecond
	if prettyPrint {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly})
	}
	zerolog.DefaultContextLogger = &log.Logger
	return nil
}

func getWhitelist() []string {
	hardcodedWhitelist := []string{
		"radix-api",
		"radix-public-site",
		"radix-web-console",
		"radix-vulnerability-scanner",
		"radix-github-webhook",
		"radix-canary-golang",
		"radix-vulnerability-scanner-api",
		"radix-servicenow-proxy",
		"radix-networkpolicy-canary",
		"radix-cost-allocation-api",
		"radix-log-api",
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
			log.Fatal().Err(err).Msg("getClusterConfig InClusterConfig")
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("getClusterConfig k8s client")
	}

	radixClient, err := radixclient.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("getClusterConfig radix client")
	}

	log.Info().Msgf("Successfully constructed k8s client to API server %v", config.Host)
	return client, radixClient
}

func getKubeUtil() (*kube.Kube, error) {
	kubeClient, radixClient := getKubernetesClient()
	kubeutil, err := kube.New(kubeClient, radixClient, nil, nil)
	if err != nil {
		return nil, err
	}
	return kubeutil, nil
}

func runFunctionPeriodically(ctx context.Context, someFunc func(ctx context.Context) error) error {
	logger := log.Ctx(ctx)
	cleanupDays, cleanupDaysErr := rootCmd.Flags().GetStringSlice(settings.CleanUpDaysOption)
	cleanupStart, cleanupStartErr := rootCmd.Flags().GetString(settings.CleanUpStartOption)
	cleanupEnd, cleanupEndErr := rootCmd.Flags().GetString(settings.CleanUpEndOption)
	period, periodErr := rootCmd.Flags().GetDuration(settings.CleanUpPeriodOption)
	err := errors.Join(cleanupDaysErr, cleanupStartErr, cleanupEndErr, periodErr)
	if err != nil {
		return err
	}
	timezone := "Local"
	window, err := timewindow.New(cleanupDays, cleanupStart, cleanupEnd, timezone)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to build time window")
	}
	source := rand.NewSource(time.Now().UnixNano())
	tick := delaytick.New(source, period)
	for range tick {
		pointInTime := time.Now()
		if window.Contains(pointInTime) {
			logger.Info().Msgf("Start listing RRs for stop %s", pointInTime)
			err := someFunc(ctx)
			if err != nil {
				return err
			}
		} else {
			logger.Info().Msgf("%s is outside of window. Continue sleeping", pointInTime)
		}
	}
	logger.Warn().Msgf("execution reached code which was presumably after an inescapable loop")
	return nil
}

func getTooInactiveRrs(ctx context.Context, kubeClient *kube.Kube, inactivityLimit time.Duration, action string) ([]v1.RadixRegistration, error) {
	rrs, err := kubeClient.ListRegistrations(ctx)
	if err != nil {
		return nil, err
	}
	var rrsForDeletion []v1.RadixRegistration
	for _, rr := range rrs {
		logger := log.Ctx(ctx).With().Str("appName", rr.Name).Logger()
		ctx := logger.WithContext(ctx)

		if isWhitelisted(rr) {
			logger.Debug().Msg("RadixRegistration is whitelisted, skipping")
			continue
		}
		ra, err := getRadixApplication(ctx, kubeClient, rr.Name)
		if kubeerrors.IsNotFound(err) {
			logger.Debug().Msg("could not find RadixApplication, continuing...")
			continue
		}
		if err != nil {
			return nil, err
		}
		namespaces := getRuntimeNamespaces(ra)
		logger.Debug().Msgf("found namespaces %s associated with RadixRegistration", strings.Join(namespaces, ", "))
		rdsForRr, err := getRadixDeploymentsInNamespaces(ctx, kubeClient, namespaces)
		if err != nil {
			return nil, err
		}
		logger.Debug().Msgf("RadixRegistration has %d RadixDeployments", len(rdsForRr))
		rjsForRr, err := getRadixJobsInNamespace(ctx, kubeClient, utils.GetAppNamespace(rr.Name))
		logger.Debug().Msgf("RadixRegistration has %d RadixJobs", len(rdsForRr))
		if err != nil {
			return nil, err
		}

		logger.Debug().Msg("Checking timestamps of RadixDeployments and RadixJobs")
		isInactive, err := rrIsInactive(ctx, rr.CreationTimestamp, rdsForRr, rjsForRr, inactivityLimit, action)
		if err != nil {
			return nil, err
		}
		if isInactive {
			rrsForDeletion = append(rrsForDeletion, *rr)
		}
	}
	return rrsForDeletion, nil
}

func getRadixJobsInNamespace(ctx context.Context, kubeClient *kube.Kube, namespace string) ([]v1.RadixJob, error) {
	rjs, err := kubeClient.RadixClient().RadixV1().RadixJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rjs.Items, nil
}

func getRadixDeploymentsInNamespaces(ctx context.Context, kubeClient *kube.Kube, namespaces []string) ([]v1.RadixDeployment, error) {
	rdsForRr := make([]v1.RadixDeployment, 0)
	for _, ns := range namespaces {
		rds, err := kubeClient.RadixClient().RadixV1().RadixDeployments(ns).List(ctx, metav1.ListOptions{})
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

func getRadixApplication(ctx context.Context, kubeClient *kube.Kube, appName string) (*v1.RadixApplication, error) {
	return kubeClient.RadixClient().RadixV1().RadixApplications(utils.GetAppNamespace(appName)).Get(ctx, appName, metav1.GetOptions{})
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

func rrIsInactive(ctx context.Context, rrCreationTimestamp metav1.Time, rds []v1.RadixDeployment, rjs []v1.RadixJob, inactivityLimit time.Duration, action string) (bool, error) {
	logger := log.Ctx(ctx)
	if rrCreationTimestamp.Add(inactivityLimit).After(time.Now()) {
		logger.Debug().Msgf("RadixRegistration is newer than inactivity limit, assuming active")
		return false, nil
	}

	if len(rds) == 0 {
		logger.Debug().Msgf("no RadixDeployments found, assuming RadixRegistration is inactive")
		return true, nil
	}

	latestRadixDeployment := SortDeploymentsByActiveFromTimestampAsc(rds)[len(rds)-1]
	latestRadixDeploymentTimestamp := latestRadixDeployment.Status.ActiveFrom
	logger.Debug().Msgf("most recent radixDeployment is %s, active from %s, %d hours ago", latestRadixDeployment.Name, latestRadixDeploymentTimestamp.Format(time.RFC822), int(time.Since(latestRadixDeploymentTimestamp.Time).Hours()))

	latestRadixJobTimestamp := metav1.Time{Time: time.Unix(0, 0)}
	latestRadixJob := getLatestRadixJob(rjs)
	if latestRadixJob != nil {
		latestRadixJobTimestamp = *latestRadixJob.Status.Created
		logger.Debug().Msgf("most recent radixJob was %s, created %s, %d hours ago", latestRadixJob.Name, latestRadixJobTimestamp.Format(time.RFC822), int(time.Since(latestRadixJobTimestamp.Time).Hours()))
	}

	latestUserMutationTimestamp, err := getLastUserMutationTimestamp(latestRadixDeployment)
	if err != nil {
		return false, err
	}

	logger.Debug().Msgf("most recent manual user activity was %s, %d hours ago", latestUserMutationTimestamp.Format(time.RFC822), int(time.Since(latestUserMutationTimestamp.Time).Hours()))
	logger.Debug().Msgf("most recent creation of RR was %s, %d hours ago", rrCreationTimestamp, int(time.Since(rrCreationTimestamp.Time).Hours()))
	lastActivity := getMostRecentTimestamp(&latestRadixJobTimestamp, latestUserMutationTimestamp, &latestRadixDeploymentTimestamp, &rrCreationTimestamp)
	logger.Debug().Msgf("lastActivity was %s, %d hours ago", lastActivity, int(time.Since(lastActivity.Time).Hours()))
	if tooLongInactivity(lastActivity, inactivityLimit) {
		logger.Debug().Msgf("last activity was %d hours ago, which is more than %d hours ago, marking for %s", int(time.Since(lastActivity.Time).Hours()), int(inactivityLimit.Hours()), action)
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
	target := make([]v1.RadixDeployment, len(rds))
	copy(target, rds)
	sort.Slice(target, func(i, j int) bool {
		return isRD1ActiveAfterRD2(&target[j], &target[i])
	})
	return target
}

func isRD1ActiveAfterRD2(rd1 *v1.RadixDeployment, rd2 *v1.RadixDeployment) bool {
	rj1ActiveFrom := rd1.Status.ActiveFrom
	rj2ActiveFrom := rd2.Status.ActiveFrom
	return rj2ActiveFrom.Before(&rj1ActiveFrom)
}

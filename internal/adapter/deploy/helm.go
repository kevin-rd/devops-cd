package deploy

import (
	"context"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/repository"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type HelmDeployer struct {
	configRepository *repository.ConfigRepository
}

const ConfigKeyChartRepoURL = "helm.repo.url"
const ConfigKeyChartRepoUsername = "helm.repo.username"
const ConfigKeyChartRepoPassword = "helm.repo.password"

var settings = cli.New()

func NewHelmDeployer(configRepository *repository.ConfigRepository) *HelmDeployer {
	return &HelmDeployer{
		configRepository: configRepository,
	}
}

// Deploy install or upgrade a chart to kubernetes, 不处理chart的依赖关系
func (d *HelmDeployer) Deploy(ctx context.Context, param *DeploymentParam) error {
	restClientGetter, err := NewRESTClientGetter(param.Kubeconfig, param.Namespace)
	if err != nil {
		return err
	}

	// 1. 初始化action config
	actionConfig := new(action.Configuration)
	if err = actionConfig.Init(restClientGetter, param.Namespace, "secret", logger.Sugar().Debugf); err != nil {
		return err
	}

	// 2.1 加载chart
	ch, err := d.loadChart(param)
	if err != nil {
		return err
	}

	// 2.2 取已合并后的values.yaml
	vals := param.Values

	// 3. upgrade or install
	var rel *release.Release

	historyClient := action.NewHistory(actionConfig)
	historyClient.Max = 1
	versions, err := historyClient.Run(param.ReleaseName)
	if err == driver.ErrReleaseNotFound || (len(versions) > 0 && versions[len(versions)-1].Info.Status == release.StatusUninstalled) {
		// If a release does not exist, we need to install it.
		client := action.NewInstall(actionConfig)
		client.Namespace = param.Namespace
		client.ReleaseName = param.ReleaseName
		//client.Atomic = true

		rel, err = client.RunWithContext(ctx, ch, vals)
		if err != nil {
			return err
		}
	} else {
		// else, upgrade it
		client := action.NewUpgrade(actionConfig)
		client.Namespace = param.Namespace
		//client.Atomic = true

		rel, err = client.RunWithContext(ctx, param.ReleaseName, ch, vals)
		if err != nil {
			return err
		}
	}

	log := logger.SugarWith(zap.String("release_name", rel.Name), zap.String("release_namespace", rel.Namespace),
		zap.Any("manifest", rel.Manifest))
	log.Debugf("Helm 部署成功! Release %s has been upgraded. Revision: %d Status:%v", rel.Name, rel.Version, rel.Info.Status)

	return nil
}

func (d *HelmDeployer) CheckStatus(ctx context.Context, param *DeploymentParam) (string, error) {
	// todo
	return "", nil
}

func (d *HelmDeployer) loadChart(param *DeploymentParam) (*chart.Chart, error) {
	url, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoURL)
	username, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoUsername)
	password, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoPassword)

	chartPathOptions := action.ChartPathOptions{
		RepoURL:  url,
		Username: username,
		Password: password,
	}

	// 更新repo
	if _, err := d.updateRepo(); err != nil {
		return nil, err
	}

	// 加载chart
	chartPath, err := chartPathOptions.LocateChart(param.AppType, settings)
	if err != nil {
		return nil, err
	}
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (d *HelmDeployer) updateRepo() (string, error) {
	url, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoURL)
	username, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoUsername)
	password, _ := d.configRepository.GetConfig(0, ConfigKeyChartRepoPassword)

	repoEntry := &repo.Entry{
		Name:     "devops-cd",
		URL:      url,
		Username: username,
		Password: password,
	}

	providers := getter.All(settings)
	r, err := repo.NewChartRepository(repoEntry, providers)
	if err != nil {
		return "", err
	}
	if _, err = r.DownloadIndexFile(); err != nil {
		return "", err
	}
	return "", nil
}

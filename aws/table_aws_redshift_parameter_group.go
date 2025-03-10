package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"
)

func tableAwsRedshiftParameterGroup(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_redshift_parameter_group",
		Description: "AWS Redshift Parameter Group",
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.SingleColumn("name"),
			ShouldIgnoreError: isNotFoundError([]string{"ClusterParameterGroupNotFound"}),
			Hydrate:           getAwsRedshiftParameterGroup,
		},
		List: &plugin.ListConfig{
			Hydrate: listAwsRedshiftParameterGroups,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "name",
				Description: "The name of the cluster parameter group.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ParameterGroupName"),
			},
			{
				Name:        "description",
				Description: "The description of the parameter group.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "family",
				Description: "The name of the cluster parameter group family that this cluster parameter group is compatible with.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ParameterGroupFamily"),
			},
			{
				Name:        "parameters",
				Description: "A list of Parameter instances. Each instance lists the parameters of one cluster parameter group.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getAwsRedshiftParameters,
			},
			{
				Name:        "tags_src",
				Description: "A list of tags assigned to the parameter group.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags"),
			},

			// Standard columns for all tables
			{
				Name:        "title",
				Description: resourceInterfaceDescription("title"),
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ParameterGroupName"),
			},
			{
				Name:        "tags",
				Description: resourceInterfaceDescription("tags"),
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags").Transform(tagListToTurbotTags),
			},
			{
				Name:        "akas",
				Description: resourceInterfaceDescription("akas"),
				Type:        proto.ColumnType_JSON,
				Hydrate:     getAwsRedshiftParameterGroupAkas,
				Transform:   transform.FromValue(),
			},
		}),
	}
}

//// LIST FUNCTION

func listAwsRedshiftParameterGroups(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	
	// TODO put me in helper function
	var region string
	matrixRegion := plugin.GetMatrixItem(ctx)[matrixKeyRegion]
	if matrixRegion != nil {
		region = matrixRegion.(string)
	}
	plugin.Logger(ctx).Trace("listAwsRedshiftParameterGroups", "AWS_REGION", region)

	// Create session
	svc, err := RedshiftService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// List call
	err = svc.DescribeClusterParameterGroupsPages(
		&redshift.DescribeClusterParameterGroupsInput{},
		func(page *redshift.DescribeClusterParameterGroupsOutput, isLast bool) bool {
			for _, parameter := range page.ParameterGroups {
				d.StreamListItem(ctx, parameter)

			}
			return !isLast
		},
	)

	return nil, err
}

//// HYDRATE FUNCTIONS

func getAwsRedshiftParameterGroup(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("getAwsRedshiftParameterGroup")

	// TODO put me in helper function
	var region string
	matrixRegion := plugin.GetMatrixItem(ctx)[matrixKeyRegion]
	if matrixRegion != nil {
		region = matrixRegion.(string)
	}

	// Create Session
	svc, err := RedshiftService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	name := d.KeyColumnQuals["name"].GetStringValue()

	// Build the params
	params := &redshift.DescribeClusterParameterGroupsInput{
		ParameterGroupName: aws.String(name),
	}

	// Get call
	data, err := svc.DescribeClusterParameterGroups(params)
	if err != nil {
		return nil, err
	}

	if data.ParameterGroups != nil && len(data.ParameterGroups) > 0 {
		return data.ParameterGroups[0], nil
	}
	return nil, nil
}

func getAwsRedshiftParameters(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("getAwsRedshiftParameters")

	// TODO put me in helper function
	var region string
	matrixRegion := plugin.GetMatrixItem(ctx)[matrixKeyRegion]
	if matrixRegion != nil {
		region = matrixRegion.(string)
	}

	// Create Session
	svc, err := RedshiftService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	name := h.Item.(*redshift.ClusterParameterGroup).ParameterGroupName

	// Build the params
	params := &redshift.DescribeClusterParametersInput{
		ParameterGroupName: name,
	}

	// Get call
	op, err := svc.DescribeClusterParameters(params)
	if err != nil {
		return nil, err
	}

	return op, nil
}

func getAwsRedshiftParameterGroupAkas(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getAwsRedshiftParameterGroupAkas")
	parameterData := h.Item.(*redshift.ClusterParameterGroup)
	c, err := getCommonColumns(ctx, d, h)
	if err != nil {
		return nil, err
	}
	commonColumnData := c.(*awsCommonColumnData)
	aka := "arn:" + commonColumnData.Partition + ":redshift:" + commonColumnData.Region + ":" + commonColumnData.AccountId + ":parametergroup"

	if strings.HasPrefix(*parameterData.ParameterGroupName, ":") {
		aka = aka + *parameterData.ParameterGroupName
	} else {
		aka = aka + ":" + *parameterData.ParameterGroupName
	}

	return []string{aka}, nil
}

//// TRANSFORM FUNCTIONS

func tagListToTurbotTags(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	plugin.Logger(ctx).Trace("tagListToTurbotTags")

	tagList := d.HydrateItem.(*redshift.ClusterParameterGroup)

	// Mapping the resource tags inside turbotTags
	var turbotTagsMap map[string]string
	if tagList != nil {
		turbotTagsMap = map[string]string{}
		for _, i := range tagList.Tags {
			turbotTagsMap[*i.Key] = *i.Value
		}
	}

	return turbotTagsMap, nil
}

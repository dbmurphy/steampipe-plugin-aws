package aws

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/cloudcontrolapi"
	"github.com/turbot/go-kit/types"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
)

func tableAwsCloudControlResource(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_cloudcontrol_resource",
		Description: "AWS Cloud Control Resource",
		List: &plugin.ListConfig{
			KeyColumns: []*plugin.KeyColumn{
				{Name: "type_name"},
				{Name: "resource_model", Require: plugin.Optional},
			},
			Hydrate: listCloudControlResources,
		},
		Get: &plugin.GetConfig{
			KeyColumns: []*plugin.KeyColumn{
				{Name: "type_name"},
				{Name: "identifier"},
			},
			Hydrate: getCloudControlResource,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "type_name",
				Description: "The name of the resource type.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromQual("type_name"),
			},
			{
				Name:        "identifier",
				Description: "The identifier for the resource.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "resource_model",
				Description: "The resource model to use to select the resources to return.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromQual("resource_model"),
			},
			{
				Name:        "properties",
				Description: "Represents information about a provisioned resource.",
				Type:        proto.ColumnType_JSON,
			},
		}),
	}
}

type cloudControlResource struct {
	Identifier *string
	Properties map[string]interface{}
}

//// LIST FUNCTION

func listCloudControlResources(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("listCloudControlResources")

	// Create session
	svc, err := CloudControlService(ctx, d)
	if err != nil {
		return nil, err
	}

	typeName := d.KeyColumnQuals["type_name"].GetStringValue()
	resourceModel := d.KeyColumnQuals["resource_model"].GetStringValue()
	input := cloudcontrolapi.ListResourcesInput{TypeName: types.String(typeName)}

	if len(resourceModel) > 0 {
		input.ResourceModel = types.String(resourceModel)
	}

	err = svc.ListResourcesPages(&input,
		func(page *cloudcontrolapi.ListResourcesOutput, isLast bool) bool {
			for _, resource := range page.ResourceDescriptions {
				identifier := resource.Identifier
				properties := resource.Properties
				var jsonMap map[string]interface{}
				unmarshalErr := json.Unmarshal([]byte(*properties), &jsonMap)
				if unmarshalErr != nil {
					plugin.Logger(ctx).Error("listCloudControlResources", "Unmarshal_error", unmarshalErr)
					panic(unmarshalErr)
				}

				d.StreamListItem(ctx, &cloudControlResource{
					Identifier: identifier,
					Properties: jsonMap,
				})

				// Check if context has been cancelled or if the limit has been hit (if specified)
				if d.QueryStatus.RowsRemaining(ctx) == 0 {
					return false
				}
			}
			return !isLast
		},
	)

	return nil, err
}

//// HYDRATE FUNCTIONS

func getCloudControlResource(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getCloudControlResource")

	// Create session
	svc, err := CloudControlService(ctx, d)
	if err != nil {
		return nil, err
	}

	typeName := d.KeyColumnQuals["type_name"].GetStringValue()
	identifier := d.KeyColumnQuals["identifier"].GetStringValue()

	input := &cloudcontrolapi.GetResourceInput{
		Identifier: types.String(identifier),
		TypeName:   types.String(typeName),
	}

	item, err := svc.GetResource(input)
	if err != nil {
		return nil, err
	}

	properties := item.ResourceDescription.Properties
	var jsonMap map[string]interface{}
	err = json.Unmarshal([]byte(*properties), &jsonMap)
	if err != nil {
		plugin.Logger(ctx).Error("getCloudControlResource", "Unmarshal_error", err)
		panic(err)
	}

	return &cloudControlResource{
		Identifier: types.String(identifier),
		Properties: jsonMap,
	}, nil

	//return jsonMap, nil
}
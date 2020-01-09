package aws

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func decodeEc2TransitGatewayRouteID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_DESTINATION", id)
	}

	return parts[0], parts[1], nil
}

func decodeEc2TransitGatewayRouteTableAssociationID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_tgw-attach-ID", id)
	}

	return parts[0], parts[1], nil
}

func decodeEc2TransitGatewayRouteTablePropagationID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_tgw-attach-ID", id)
	}

	return parts[0], parts[1], nil
}

func ec2DescribeTransitGateway(conn *ec2.EC2, transitGatewayID string) (*ec2.TransitGateway, error) {
	input := &ec2.DescribeTransitGatewaysInput{
		TransitGatewayIds: []*string{aws.String(transitGatewayID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway (%s): %s", transitGatewayID, input)
	for {
		output, err := conn.DescribeTransitGateways(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGateways) == 0 {
			return nil, nil
		}

		for _, transitGateway := range output.TransitGateways {
			if transitGateway == nil {
				continue
			}

			if aws.StringValue(transitGateway.TransitGatewayId) == transitGatewayID {
				return transitGateway, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRoute(conn *ec2.EC2, transitGatewayRouteTableID, destination string) (*ec2.TransitGatewayRoute, error) {
	input := &ec2.SearchTransitGatewayRoutesInput{
		// As of the time of writing, the EC2 API reference documentation (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SearchTransitGatewayRoutes.html)
		// incorrectly states which filter Names are allowed. The below are example errors:
		// InvalidParameterValue: Value (transit-gateway-route-destination-cidr-block) for parameter Filters is invalid.
		// InvalidParameterValue: Value (transit-gateway-route-type) for parameter Filters is invalid.
		// InvalidParameterValue: Value (destination-cidr-block) for parameter Filters is invalid.
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("type"),
				Values: []*string{aws.String("static")},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Searching EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, input)
	output, err := conn.SearchTransitGatewayRoutes(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Routes) == 0 {
		return nil, nil
	}

	for _, route := range output.Routes {
		if route == nil {
			continue
		}

		if aws.StringValue(route.DestinationCidrBlock) == destination {
			return route, nil
		}
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRouteTable(conn *ec2.EC2, transitGatewayRouteTableID string) (*ec2.TransitGatewayRouteTable, error) {
	input := &ec2.DescribeTransitGatewayRouteTablesInput{
		TransitGatewayRouteTableIds: []*string{aws.String(transitGatewayRouteTableID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, input)
	for {
		output, err := conn.DescribeTransitGatewayRouteTables(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGatewayRouteTables) == 0 {
			return nil, nil
		}

		for _, transitGatewayRouteTable := range output.TransitGatewayRouteTables {
			if transitGatewayRouteTable == nil {
				continue
			}

			if aws.StringValue(transitGatewayRouteTable.TransitGatewayRouteTableId) == transitGatewayRouteTableID {
				return transitGatewayRouteTable, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRouteTableAssociation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) (*ec2.TransitGatewayRouteTableAssociation, error) {
	if transitGatewayRouteTableID == "" {
		return nil, nil
	}

	input := &ec2.GetTransitGatewayRouteTableAssociationsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("transit-gateway-attachment-id"),
				Values: []*string{aws.String(transitGatewayAttachmentID)},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	output, err := conn.GetTransitGatewayRouteTableAssociations(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Associations) == 0 {
		return nil, nil
	}

	return output.Associations[0], nil
}

func ec2DescribeTransitGatewayRouteTablePropagation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) (*ec2.TransitGatewayRouteTablePropagation, error) {
	if transitGatewayRouteTableID == "" {
		return nil, nil
	}

	input := &ec2.GetTransitGatewayRouteTablePropagationsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("transit-gateway-attachment-id"),
				Values: []*string{aws.String(transitGatewayAttachmentID)},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	output, err := conn.GetTransitGatewayRouteTablePropagations(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.TransitGatewayRouteTablePropagations) == 0 {
		return nil, nil
	}

	return output.TransitGatewayRouteTablePropagations[0], nil
}

func ec2DescribeTransitGatewayPeeringAttachment(conn *ec2.EC2, transitGatewayAttachmentID string) (*ec2.TransitGatewayPeeringAttachment, error) {
	input := &ec2.DescribeTransitGatewayPeeringAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{aws.String(transitGatewayAttachmentID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway Peering Attachment (%s): %s", transitGatewayAttachmentID, input)
	for {
		output, err := conn.DescribeTransitGatewayPeeringAttachments(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGatewayPeeringAttachments) == 0 {
			return nil, nil
		}

		for _, transitGatewayPeeringAttachment := range output.TransitGatewayPeeringAttachments {
			if transitGatewayPeeringAttachment == nil {
				continue
			}

			if aws.StringValue(transitGatewayPeeringAttachment.TransitGatewayAttachmentId) == transitGatewayAttachmentID {
				return transitGatewayPeeringAttachment, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayVpcAttachment(conn *ec2.EC2, transitGatewayAttachmentID string) (*ec2.TransitGatewayVpcAttachment, error) {
	input := &ec2.DescribeTransitGatewayVpcAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{aws.String(transitGatewayAttachmentID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway VPC Attachment (%s): %s", transitGatewayAttachmentID, input)
	for {
		output, err := conn.DescribeTransitGatewayVpcAttachments(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGatewayVpcAttachments) == 0 {
			return nil, nil
		}

		for _, transitGatewayVpcAttachment := range output.TransitGatewayVpcAttachments {
			if transitGatewayVpcAttachment == nil {
				continue
			}

			if aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId) == transitGatewayAttachmentID {
				return transitGatewayVpcAttachment, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayMulticastDomain(conn *ec2.EC2, domainID string) (*ec2.TransitGatewayMulticastDomain, error) {
	if conn == nil || domainID == "" {
		return nil, nil
	}

	input := &ec2.DescribeTransitGatewayMulticastDomainsInput{
		// Note: one or more filters required
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("transit-gateway-multicast-domain-id"),
				Values: []*string{aws.String(domainID)},
			},
		},
		TransitGatewayMulticastDomainIds: []*string{aws.String(domainID)},
	}

	output, err := conn.DescribeTransitGatewayMulticastDomains(input)
	if err != nil {
		return nil, err
	}

	if output == nil || len(output.TransitGatewayMulticastDomains) == 0 {
		return nil, nil
	}

	return output.TransitGatewayMulticastDomains[0], nil
}

func ec2GetTransitGatewayMulticastDomainAssociations(conn *ec2.EC2, domainID string) ([]*ec2.TransitGatewayMulticastDomainAssociation, error) {
	if conn == nil || domainID == "" {
		return nil, nil
	}

	input := &ec2.GetTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
	}

	var associations []*ec2.TransitGatewayMulticastDomainAssociation
	log.Printf("[DEBUG] Reading EC2 Transit Gateway Multicast Domain (%s) Associations: %s", domainID, input)
	for {
		output, err := conn.GetTransitGatewayMulticastDomainAssociations(input)
		if err != nil {
			return nil, err
		}

		if output == nil {
			return nil, nil
		}

		for _, association := range output.MulticastDomainAssociations {
			associations = append(associations, association)
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}

	return associations, nil
}

func ec2SearchTransitGatewayMulticastDomainGroups(conn *ec2.EC2, domainID string, filters []*ec2.Filter) ([]*ec2.TransitGatewayMulticastGroup, error) {
	if conn == nil || domainID == "" {
		return nil, nil
	}

	input := &ec2.SearchTransitGatewayMulticastGroupsInput{
		Filters:                         filters,
		TransitGatewayMulticastDomainId: aws.String(domainID),
	}

	var groups []*ec2.TransitGatewayMulticastGroup
	log.Printf("[DEBUG] Reading EC2 Transit Gateway Multicast Domain (%s) groups: %s", domainID, input)
	for {
		output, err := conn.SearchTransitGatewayMulticastGroups(input)
		if err != nil {
			return nil, err
		}

		if output == nil {
			return nil, nil
		}

		for _, group := range output.MulticastGroups {
			groups = append(groups, group)
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}

	return groups, nil
}

func ec2SearchTransitGatewayMulticastDomainGroupsByType(conn *ec2.EC2, domainID string, member bool) ([]*ec2.TransitGatewayMulticastGroup, error) {
	return ec2SearchTransitGatewayMulticastDomainGroups(conn, domainID, ec2SearchTransitGatewayMulticastDomainGroupsTypeFilter(member))
}

func ec2SearchTransitGatewayMulticastDomainGroupsTypeFilter(member bool) []*ec2.Filter {
	var filters []*ec2.Filter
	if member {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("is-group-member"),
			Values: []*string{aws.String("true")},
		})
	} else {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("is-group-source"),
			Values: []*string{aws.String("true")},
		})
	}
	return filters
}

func ec2SearchTransitGatewayMulticastDomainGroupIpFilters(member bool, groupIP string) []*ec2.Filter {
	return append(ec2SearchTransitGatewayMulticastDomainGroupsTypeFilter(member), &ec2.Filter{
		Name:   aws.String("group-ip-address"),
		Values: []*string{aws.String(groupIP)},
	})
}

func ec2TransitGatewayRouteTableAssociationUpdate(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string, associate bool) error {
	transitGatewayAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	if associate && transitGatewayAssociation == nil {
		input := &ec2.AssociateTransitGatewayRouteTableInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.AssociateTransitGatewayRouteTable(input); err != nil {
			return fmt.Errorf("error associating EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if err := waitForEc2TransitGatewayRouteTableAssociationCreation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
			return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}
	} else if !associate && transitGatewayAssociation != nil {
		input := &ec2.DisassociateTransitGatewayRouteTableInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.DisassociateTransitGatewayRouteTable(input); err != nil {
			return fmt.Errorf("error disassociating EC2 Transit Gateway Route Table (%s) disassociation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if err := waitForEc2TransitGatewayRouteTableAssociationDeletion(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
			return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) disassociation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}
	}

	return nil
}

func ec2TransitGatewayRouteTablePropagationUpdate(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string, enablePropagation bool) error {
	transitGatewayRouteTablePropagation, err := ec2DescribeTransitGatewayRouteTablePropagation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
	}

	if enablePropagation && transitGatewayRouteTablePropagation == nil {
		input := &ec2.EnableTransitGatewayRouteTablePropagationInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.EnableTransitGatewayRouteTablePropagation(input); err != nil {
			return fmt.Errorf("error enabling EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
		}
	} else if !enablePropagation && transitGatewayRouteTablePropagation != nil {
		input := &ec2.DisableTransitGatewayRouteTablePropagationInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.DisableTransitGatewayRouteTablePropagation(input); err != nil {
			return fmt.Errorf("error disabling EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
		}
	}

	return nil
}

func ec2TransitGatewayRefreshFunc(conn *ec2.EC2, transitGatewayID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)

		if isAWSErr(err, "InvalidTransitGatewayID.NotFound", "") {
			return nil, ec2.TransitGatewayStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway (%s): %s", transitGatewayID, err)
		}

		if transitGateway == nil {
			return nil, ec2.TransitGatewayStateDeleted, nil
		}

		return transitGateway, aws.StringValue(transitGateway.State), nil
	}
}

func ec2TransitGatewayRouteTableRefreshFunc(conn *ec2.EC2, transitGatewayRouteTableID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayRouteTable, err := ec2DescribeTransitGatewayRouteTable(conn, transitGatewayRouteTableID)

		if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, err)
		}

		if transitGatewayRouteTable == nil {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		return transitGatewayRouteTable, aws.StringValue(transitGatewayRouteTable.State), nil
	}
}

func ec2TransitGatewayRouteTableAssociationRefreshFunc(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)

		if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s) Association for (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if transitGatewayAssociation == nil {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		return transitGatewayAssociation, aws.StringValue(transitGatewayAssociation.State), nil
	}
}

func ec2TransitGatewayPeeringAttachmentRefreshFunc(conn *ec2.EC2, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayPeeringAttachment, err := ec2DescribeTransitGatewayPeeringAttachment(conn, transitGatewayAttachmentID)

		if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Peering Attachment (%s): %s", transitGatewayAttachmentID, err)
		}

		if transitGatewayPeeringAttachment == nil {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		if aws.StringValue(transitGatewayPeeringAttachment.State) == ec2.TransitGatewayAttachmentStateFailed && transitGatewayPeeringAttachment.Status != nil {
			return transitGatewayPeeringAttachment, aws.StringValue(transitGatewayPeeringAttachment.State), fmt.Errorf("%s: %s", aws.StringValue(transitGatewayPeeringAttachment.Status.Code), aws.StringValue(transitGatewayPeeringAttachment.Status.Message))
		}

		return transitGatewayPeeringAttachment, aws.StringValue(transitGatewayPeeringAttachment.State), nil
	}
}

func ec2TransitGatewayVpcAttachmentRefreshFunc(conn *ec2.EC2, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayVpcAttachment, err := ec2DescribeTransitGatewayVpcAttachment(conn, transitGatewayAttachmentID)

		if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway VPC Attachment (%s): %s", transitGatewayAttachmentID, err)
		}

		if transitGatewayVpcAttachment == nil {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		return transitGatewayVpcAttachment, aws.StringValue(transitGatewayVpcAttachment.State), nil
	}
}

func ec2TransitGatewayMulticastDomainRefreshFunc(conn *ec2.EC2, domainID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		multicastDomain, err := ec2DescribeTransitGatewayMulticastDomain(conn, domainID)
		if isAWSErr(err, "InvalidTransitGatewayMulticastDomainId.NotFound", "") {
			return nil, ec2.TransitGatewayMulticastDomainStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Multicast Domain (%s): %s", domainID, err)
		}

		if multicastDomain == nil {
			return nil, ec2.TransitGatewayMulticastDomainStateDeleted, nil
		}

		return multicastDomain, aws.StringValue(multicastDomain.State), nil
	}
}

func ec2TransitGatewayMulticastDomainAssociationRefreshFunc(conn *ec2.EC2, domainID string, subnetIDs []*string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		associations, err := ec2GetTransitGatewayMulticastDomainAssociations(conn, domainID)
		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Multicast Domain associations: %s", err)
		}

		subnetStates := make(map[string]string)
		for _, subnetID := range subnetIDs {
			subnetStates[aws.StringValue(subnetID)] = ""
		}

		for _, association := range associations {
			if association == nil {
				continue
			}
			subnet := association.Subnet
			subnetID := aws.StringValue(subnet.SubnetId)
			if _, exists := subnetStates[subnetID]; exists {
				subnetStates[subnetID] = aws.StringValue(subnet.State)
				continue
			}
		}

		for subnetID, subnetState := range subnetStates {
			if subnetState == "" {
				// Not found, mark as functionally disassociated
				subnetStates[subnetID] = ec2.AssociationStatusCodeDisassociated
			}
		}

		log.Printf(
			"[DEBUG] Current EC2 Transit Gateway Multicast Domain (%s) states:\n\t%s", domainID, subnetStates)

		// Note: Since we are potentially associating/disassociating multiple subnets here, we will have this refresh
		// function only return "associated" once all of the subnets are associated or "disassociated" once all
		// disassociated
		// if we encounter anything else, return immediately
		// if we encounter mixed "disassociated" and "associated", raise an error
		compoundState := ""
		for _, state := range subnetStates {
			if compoundState == "" {
				compoundState = state
				continue
			}
			switch state {
			case ec2.AssociationStatusCodeAssociationFailed:
			case ec2.AssociationStatusCodeDisassociating:
			case ec2.AssociationStatusCodeAssociating:
				return associations, state, nil
			case ec2.AssociationStatusCodeDisassociated:
			case ec2.AssociationStatusCodeAssociated:
				if compoundState != state {
					return nil, "", fmt.Errorf("received conflicting association states")
				}
			default:
				return nil, "", fmt.Errorf("unhandled association state: %s", state)
			}
		}

		return associations, compoundState, nil
	}
}

func waitForEc2TransitGatewayCreation(conn *ec2.EC2, transitGatewayID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayStatePending},
		Target:  []string{ec2.TransitGatewayStateAvailable},
		Refresh: ec2TransitGatewayRefreshFunc(conn, transitGatewayID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway (%s) availability", transitGatewayID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayDeletion(conn *ec2.EC2, transitGatewayID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayStateAvailable,
			ec2.TransitGatewayStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayStateDeleted},
		Refresh:        ec2TransitGatewayRefreshFunc(conn, transitGatewayID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway (%s) deletion", transitGatewayID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableCreation(conn *ec2.EC2, transitGatewayRouteTableID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayRouteTableStatePending},
		Target:  []string{ec2.TransitGatewayRouteTableStateAvailable},
		Refresh: ec2TransitGatewayRouteTableRefreshFunc(conn, transitGatewayRouteTableID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) availability", transitGatewayRouteTableID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayRouteTableDeletion(conn *ec2.EC2, transitGatewayRouteTableID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayRouteTableStateAvailable,
			ec2.TransitGatewayRouteTableStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayRouteTableStateDeleted},
		Refresh:        ec2TransitGatewayRouteTableRefreshFunc(conn, transitGatewayRouteTableID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) deletion", transitGatewayRouteTableID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableAssociationCreation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAssociationStateAssociating},
		Target:  []string{ec2.TransitGatewayAssociationStateAssociated},
		Refresh: ec2TransitGatewayRouteTableAssociationRefreshFunc(conn, transitGatewayRouteTableID, transitGatewayAttachmentID),
		Timeout: 5 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) association: %s", transitGatewayRouteTableID, transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayRouteTableAssociationDeletion(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAssociationStateAssociated,
			ec2.TransitGatewayAssociationStateDisassociating,
		},
		Target:         []string{""},
		Refresh:        ec2TransitGatewayRouteTableAssociationRefreshFunc(conn, transitGatewayRouteTableID, transitGatewayAttachmentID),
		Timeout:        5 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) disassociation: %s", transitGatewayRouteTableID, transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayPeeringAttachmentAcceptance(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStatePending,
			ec2.TransitGatewayAttachmentStatePendingAcceptance,
		},
		Target:  []string{ec2.TransitGatewayAttachmentStateAvailable},
		Refresh: ec2TransitGatewayPeeringAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Peering Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayPeeringAttachmentCreation(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStateFailing,
			ec2.TransitGatewayAttachmentStatePending,
			"initiatingRequest", // No ENUM currently exists in the SDK for the state given by AWS
		},
		Target: []string{
			ec2.TransitGatewayAttachmentStateAvailable,
			ec2.TransitGatewayAttachmentStatePendingAcceptance,
		},
		Refresh: ec2TransitGatewayPeeringAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Peering Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayPeeringAttachmentDeletion(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStateAvailable,
			ec2.TransitGatewayAttachmentStateDeleting,
			ec2.TransitGatewayAttachmentStatePendingAcceptance,
			ec2.TransitGatewayAttachmentStateRejected,
		},
		Target:  []string{ec2.TransitGatewayAttachmentStateDeleted},
		Refresh: ec2TransitGatewayPeeringAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Peering Attachment (%s) deletion", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayVpcAttachmentAcceptance(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStatePending,
			ec2.TransitGatewayAttachmentStatePendingAcceptance,
		},
		Target:  []string{ec2.TransitGatewayAttachmentStateAvailable},
		Refresh: ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayVpcAttachmentCreation(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAttachmentStatePending},
		Target: []string{
			ec2.TransitGatewayAttachmentStatePendingAcceptance,
			ec2.TransitGatewayAttachmentStateAvailable,
		},
		Refresh: ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayVpcAttachmentDeletion(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStateAvailable,
			ec2.TransitGatewayAttachmentStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayAttachmentStateDeleted},
		Refresh:        ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) deletion", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayVpcAttachmentUpdate(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAttachmentStateModifying},
		Target:  []string{ec2.TransitGatewayAttachmentStateAvailable},
		Refresh: ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayMulticastDomainCreation(conn *ec2.EC2, domainID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayMulticastDomainStatePending},
		Target:  []string{ec2.TransitGatewayMulticastDomainStateAvailable},
		Refresh: ec2TransitGatewayMulticastDomainRefreshFunc(conn, domainID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Multicast Domain (%s) availability", domainID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayMulticastDomainDeletion(conn *ec2.EC2, domainID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayMulticastDomainStateAvailable,
			ec2.TransitGatewayMulticastDomainStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayMulticastDomainStateDeleted},
		Refresh:        ec2TransitGatewayMulticastDomainRefreshFunc(conn, domainID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Multicast Domain (%s) deletion", domainID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayMulticastDomainAssociation(conn *ec2.EC2, domainID string, subnetIDs []*string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.AssociationStatusCodeAssociating},
		Target:  []string{ec2.AssociationStatusCodeAssociated},
		Refresh: ec2TransitGatewayMulticastDomainAssociationRefreshFunc(conn, domainID, subnetIDs),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Multicast Domain associations")
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayMulticastDomainDisassociation(conn *ec2.EC2, domainID string, subnetIDs []*string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.AssociationStatusCodeAssociated,
			ec2.AssociationStatusCodeDisassociating,
		},
		Target:         []string{ec2.AssociationStatusCodeDisassociated},
		Refresh:        ec2TransitGatewayMulticastDomainAssociationRefreshFunc(conn, domainID, subnetIDs),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Multicast Domain dissasociation(s)")
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayMulticastDomainGroupRegister(conn *ec2.EC2, domainID string, groupData map[string]interface{}, member bool) error {
	filters := ec2SearchTransitGatewayMulticastDomainGroupIpFilters(member, groupData["group_ip_address"].(string))
	netIDs := groupData["network_interface_ids"].(*schema.Set)

	log.Printf(
		"[DEBUG] Validating EC2 Transit Gateway Multicast Domain (%s) group was registered successfully",
		domainID)

	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		groups, err := ec2SearchTransitGatewayMulticastDomainGroups(conn, domainID, filters)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		// find each net ID for this group
		for _, netID := range netIDs.List() {
			found := false
			for _, group := range groups {
				if aws.StringValue(group.NetworkInterfaceId) == netID {
					found = true
					break
				}
			}

			if !found {
				return resource.RetryableError(fmt.Errorf(
					"EC2 Transit Gateway Multicast Domain (%s) group not available: %s",
					domainID, groupData))
			}
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		return fmt.Errorf(
			"error validating that EC2 Transit Gateway Multicast Domain (%s) group was successfully "+
				"registered: %s", domainID, err)
	}

	return nil
}

func waitForEc2TransitGatewayMulticastDomainGroupDeregister(conn *ec2.EC2, domainID string, groupData map[string]interface{}, member bool) error {
	filters := ec2SearchTransitGatewayMulticastDomainGroupIpFilters(member, groupData["group_ip_address"].(string))
	netIDs := groupData["network_interface_ids"].(*schema.Set)

	log.Printf(
		"[DEBUG] Validating EC2 Transit Gateway Multicast Domain (%s) group was deregistered successfully",
		domainID)

	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		groups, err := ec2SearchTransitGatewayMulticastDomainGroups(conn, domainID, filters)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		// make sure no net IDs from this group are found
		for _, netID := range netIDs.List() {
			for _, group := range groups {
				if aws.StringValue(group.NetworkInterfaceId) == netID {
					return resource.RetryableError(
						fmt.Errorf("EC2 Transit Gateway Multicast Domain (%s) still available: %s",
							domainID, groupData))
				}
			}
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		return fmt.Errorf(
			"error validating that EC2 Transit Gateway Multicast Domain (%s) group was successfully "+
				"deregistered: %s", domainID, err)
	}

	return nil
}

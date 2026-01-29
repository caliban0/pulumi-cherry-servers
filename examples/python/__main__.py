import pulumi
import caliban0_pulumi_cherry_servers as cherry

# salt = cherry.RandomSalt("salt", password="test", salted_length=6)
# pulumi.export(
#     "salt",
#     {
#         "salt": salt.salt,
#         "salted_password": salt.salted_password,
#         "password": salt.password,
#         "salt_length": salt.salted_length,
#     },
# )

project = cherry.Project("myProject", team=148226, bgp=True, name="testt")
# pulumi.export(
#     "project_json",
#     pulumi.Output.json_dumps(
#         {
#             "id": project.id,
#             "name": project.name,
#             "team": project.team,
#             "bgp": project.bgp,
#             "local_asn": project.local_asn,
#         }
#     ),
# )
# pulumi.export(
#     "project_output",
#     {
#         "id": project.id,
#         "name": project.name,
#         "team": project.team,
#         "bgp": project.bgp,
#         "local_asn": project.local_asn,
#     },
# )


ip = cherry.IP(
    "myIP",
    region="LT-Siauliai",
    project=project.id.apply(int),
    ptr_record="test-a",
    a_record="test-a",
    tags={"env": "dev"},
)
pulumi.export(
    "ip_json",
    pulumi.Output.json_dumps(
        {
            "region": ip.region,
            "project": ip.project,
            "ptrRecord": ip.ptr_record,
            "aRecord": ip.a_record,
            "routedTo": ip.routed_to,
            "targetedTo": ip.targeted_to,
            "tags": ip.tags,
            "address": ip.address,
            "address_family": ip.address_family,
            "cidr": ip.cidr,
            "type": ip.type,
            "ptrRecordEffective": ip.ptr_record_effective,
            "aRecordEffective": ip.a_record_effective,
        }
    ),
)

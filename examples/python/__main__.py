import pulumi
import caliban0_pulumi_cherry_servers as cherry

project = cherry.Project("myProject", team=148226, bgp=True)
pulumi.export(
    "output",
    {
        "bgp": project.bgp,
        "name": project.name,
        "team": project.team,
        "asn": project.local_asn,
    },
)

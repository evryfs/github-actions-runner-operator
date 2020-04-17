package com.tietoevry.fss.garo.github

import javax.ws.rs.*
import javax.ws.rs.core.MediaType

@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
@Path("/")
interface RunnerApi {

    @GET
    @Path("/orgs/{organization}/actions/runners")
    fun listOrgRunners(@HeaderParam("Authorization") token: String,
                       @PathParam("organization") organization: String): Runners
}
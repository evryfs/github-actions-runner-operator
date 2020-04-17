package com.tietoevry.fss.garo.crd

import com.fasterxml.jackson.databind.annotation.JsonDeserialize
import io.fabric8.kubernetes.api.model.OwnerReference
import io.fabric8.kubernetes.api.model.OwnerReferenceBuilder
import io.fabric8.kubernetes.client.CustomResource

@JsonDeserialize
data class ActionRunner(var spec: ActionRunnerSpec = ActionRunnerSpec(),
                        var status: ActionRunnerStatus = ActionRunnerStatus()): CustomResource() {

    fun asOwnerRef(): OwnerReference =
        OwnerReferenceBuilder()
                .withApiVersion(this.apiVersion)
                .withKind(this.kind)
                .withName(this.metadata.name)
                .withController(true)
                .withUid(this.metadata.uid)
                .build()
}
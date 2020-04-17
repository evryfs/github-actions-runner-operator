package com.tietoevry.fss.garo.crd

import com.fasterxml.jackson.databind.annotation.JsonDeserialize
import io.fabric8.kubernetes.client.CustomResourceList

@JsonDeserialize
class ActionRunnerList : CustomResourceList<ActionRunner>()
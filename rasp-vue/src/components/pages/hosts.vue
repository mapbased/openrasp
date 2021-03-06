<template>
  <div class="my-3 my-md-5">
    <div class="container">
      <div class="page-header">
        <h1 class="page-title">
          Agent 管理
        </h1>
        <div class="page-options d-flex">
          <div>
            <b-dropdown text="主机状态" class="">
              <div class="row px-2">
                <div class="col-6">
                  <label class="custom-switch">
                    <input v-model="filter.online" type="checkbox" checked="filter.online" class="custom-switch-input" @change="$emit('selected')">
                    <span class="custom-switch-indicator" />
                    <span class="custom-switch-description">
                      在线
                    </span>
                  </label>
                </div>
                <div class="col-6">
                  <label class="custom-switch">
                    <input v-model="filter.offline" type="checkbox" checked="filter.offline" class="custom-switch-input" @change="$emit('selected')">
                    <span class="custom-switch-indicator" />
                    <span class="custom-switch-description">
                      离线
                    </span>
                  </label>
                </div>
              </div>
            </b-dropdown>
          </div>
          <div class="input-icon ml-2">
            <span class="input-icon-addon">
              <i class="fe fe-search" />
            </span>
            <input v-model="hostname" type="text" class="form-control w-10" placeholder="搜索主机或者IP">
          </div>

          <button class="btn btn-primary ml-2" @click="loadRaspList(1)">
            搜索
          </button>
        </div>
      </div>
      <div class="card">
        <div class="card-body">
          <vue-loading v-if="loading" type="spiningDubbles" color="rgb(90, 193, 221)" :size="{ width: '50px', height: '50px' }" />

          <table v-if="! loading" class="table table-hover table-bordered">
            <thead>
              <tr>
                <th nowrap>
                  主机名
                </th>
                <th>
                  注册 IP
                </th>
                <th>
                  RASP 版本
                </th>
                <th>
                  RASP 目录
                </th>
                <th>
                  上次通信
                </th>
                <th>
                  状态
                </th>
                <th>
                  操作
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in data" :key="row.id">
                <td>
                  {{ row.hostname }}
                </td>
                <td nowrap>
                  {{ row.register_ip }}
                </td>
                <td nowrap>
                  {{ row.language }}/{{ row.version }} <br>
                  official/{{ row.plugin_version }}
                </td>
                <td>
                  {{ row.rasp_home }}
                </td>
                <td nowrap>
                  {{ moment(row.last_heartbeat_time * 1000).format('YYYY-MM-DD') }} <br>
                  {{ moment(row.last_heartbeat_time * 1000).format('HH:mm:ss') }}
                </td>
                <td nowrap>
                  <span v-if="! row.online" class="text-danger">
                    离线
                  </span>
                  <span v-if="row.online">
                    正常
                  </span>
                </td>
                <td nowrap>
                  <a href="javascript:" @click="doDelete(row)">
                    删除
                  </a>
                </td>
              </tr>
            </tbody>
          </table>
          <nav v-if="! loading">
            <b-pagination v-model="currentPage" align="center" :total-rows="total" :per-page="10" @change="loadRaspList($event)" />
          </nav>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import isIp from 'is-ip'

export default {
  name: 'Hosts',
  data: function() {
    return {
      data: [],
      loading: false,
      currentPage: 1,
      total: 0,
      hostname: '',
      filter: {
        online: true,
        offline: true
      }
    }
  },
  computed: {
    ...mapGetters(['current_app'])
  },
  watch: {
    current_app() { this.loadRaspList(1) },
    filter: {
      handler() { this.loadRaspList(1) },
      deep: true
    }
  },
  mounted() {
    if (!this.current_app.id) {
      return
    }
    this.loadRaspList(1)
  },
  methods: {
    loadRaspList(page) {
      if (!this.filter.online && !this.filter.offline) {
        this.currentPage = page
        this.data = []
        this.total = 0
        this.loading = false
        return
      }
      const body = {
        data: {
          app_id: this.current_app.id
        },
        page: page,
        perpage: 10
      }
      if (this.hostname) {
        if (isIp(this.hostname)) {
          body.data.register_ip = this.hostname
        } else {
          body.data.hostname = this.hostname
        }
      }
      if (this.filter.online && !this.filter.offline) {
        body.data.online = true
      } else if (!this.filter.online && this.filter.offline) {
        body.data.online = false
      }
      this.loading = true
      return this.request.post('v1/api/rasp/search', body).then(res => {
        this.currentPage = page
        this.data = res.data
        this.total = res.total
        this.loading = false
      })
    },
    doDelete: function(data) {
      if (!confirm('确认删除? 删除前请先在主机端卸载 OpenRASP agent')) {
        return
      }
      var body = {
        id: data.id
      }

      this.api_request('v1/api/rasp/delete', body, function(
        data
      ) {
        this.loadRaspList()
      })
    }
  }
}
</script>

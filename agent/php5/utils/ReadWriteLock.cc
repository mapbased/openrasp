/*
 * Copyright 2017-2018 Baidu Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "ReadWriteLock.h"

namespace openrasp
{

ReadWriteLock::ReadWriteLock(pthread_rwlock_t *rwlock, enum LOCK_TYPE lock_type)
    : rwlock(rwlock), lock_type(lock_type)
{
  if (lock_type == LOCK_PROCESS)
  {
    pthread_rwlockattr_init(&rwlock_attr);
    pthread_rwlockattr_setpshared(&rwlock_attr, PTHREAD_PROCESS_SHARED);
    pthread_rwlock_init(rwlock, &rwlock_attr);
  }
  else
  {
    pthread_rwlock_init(rwlock, nullptr);
  }
}

ReadWriteLock::~ReadWriteLock()
{
  if (lock_type == LOCK_PROCESS)
  {
    pthread_rwlockattr_destroy(&rwlock_attr);
  }
  pthread_rwlock_destroy(rwlock);
}

bool ReadWriteLock::read_lock()
{
  if (pthread_rwlock_rdlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

bool ReadWriteLock::read_unlock()
{
  if (pthread_rwlock_unlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

bool ReadWriteLock::read_try_lock()
{
  if (pthread_rwlock_tryrdlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

bool ReadWriteLock::write_lock()
{
  if (pthread_rwlock_wrlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

bool ReadWriteLock::write_unlock()
{
  if (pthread_rwlock_unlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

bool ReadWriteLock::write_try_lock()
{
  if (pthread_rwlock_trywrlock(rwlock) != 0)
  {
    return false;
  }
  return true;
}

} // namespace openrasp